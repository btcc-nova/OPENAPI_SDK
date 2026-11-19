"""All requests are explicit; no redirects, retries or automatic heartbeat."""
import hashlib
import http.client
import json
import math
from decimal import Decimal, InvalidOperation
from urllib.parse import urlencode, urlsplit


class SDKError(Exception):
    """Base class for service, protocol and transport failures."""


class APIError(SDKError):
    def __init__(self, code, message):
        self.code, self.message = code, message  # Raw service message: do not log blindly.
        super().__init__(f"BTCC API error: {code}")


class HTTPError(SDKError):
    def __init__(self, status_code):
        self.status_code = status_code
        super().__init__(f"BTCC HTTP status: {status_code}")


class ProtocolError(SDKError):
    """Response does not match the expected format."""


class TransportError(SDKError):
    """Connection failed. An outstanding trade may already have executed."""


class Session:
    """Returned by login(); pass it only to the client that created it."""
    __slots__ = ("_client", "_token", "_secret", "_account_id", "_group_id")

    def __init__(self, client, token, secret, account_id, group_id):
        self._client, self._token, self._secret = client, token, secret
        self._account_id, self._group_id = account_id, group_id


def _text(value, name):
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{name} must be a non-empty string")
    return value


def _uint(value, name, *, minimum=0, bits=64):
    if type(value) is not int or not minimum <= value < 1 << bits:
        raise ValueError(f"invalid {name}")
    return value


def _number(value):
    if isinstance(value, bool) or not isinstance(value, (int, str, Decimal)):
        raise ValueError("use Decimal, int or a decimal string (not float)")
    try:
        n = Decimal(value)
    except InvalidOperation:
        raise ValueError("invalid decimal") from None
    if not n.is_finite():
        raise ValueError("decimal must be finite")
    return n


def _amount(value, *, positive=False):
    n = _number(value)
    if n < 0 or (positive and not n):
        raise ValueError("invalid price or volume")
    # Do not expand unbounded scientific notation into a request body.
    if abs(n.as_tuple().exponent) > 8192 or n.adjusted() > 8192:
        raise ValueError("decimal too large to encode")
    if not n:
        return "0"
    s = format(n, "f")
    return s.rstrip("0").rstrip(".") if "." in s else s


# Financial fields can arrive as JSON numbers or strings. Preserve other strings,
# IDs and integer product states (including enable=3) as sent by the server.
_DECIMALS = frozenset('''Price balance equity margin free_margin margin_level margin_stopout bonus
range_min range_max margin_initial margin_maintenance margin_hedged special_margin_initial
special_margin_maintenance special_margin_stopout special_margin_hedged holiday_margin_initial
holiday_margin_maintenance holiday_margin_stopout holiday_margin_hedged commission_rebate value
volumes_min volumes_step volumes_max contract_size long_swap short_swap stop_level maxstop_level
pips_ratio spread spread_balance range_init position_volumes_max max_pos_value gap_value
margin_call_level stop_out_level close_price commission exec_commission exec_price exec_profit
exec_swap exec_volume init_volume open_price profit rate stop_loss swap take_profit volume
open_margin request_volume request_price execute_volume execute_price'''.split())


def _amounts(obj):
    if isinstance(obj, list):
        for item in obj:
            _amounts(item)
    elif isinstance(obj, dict):
        for key, value in obj.items():
            if key in _DECIMALS:
                obj[key] = _number(value)
            else:
                _amounts(value)
    return obj


def _field(obj, key, kind=dict):
    value = obj.get(key)
    if not isinstance(value, kind) or (kind is list and not all(isinstance(v, dict) for v in value)):
        raise ProtocolError(f"invalid or missing {key}")
    return value


def _url(value):
    _text(value, "URL")
    try:
        parts = urlsplit(value)
        if (parts.scheme in ("http", "https") and parts.hostname and parts.port != 0
                and parts.username is None and "?" not in value and "#" not in value
                and "\\" not in value and not any(ord(c) <= 32 or ord(c) == 127 for c in value)):
            return parts
    except ValueError:
        pass
    raise ValueError("expected HTTP(S) URL without credentials, query or fragment")


class Client:
    def __init__(self, base_url, *, quote_url=None, timeout=10):
        self._base = _url(base_url)
        self._quote = _url(quote_url) if quote_url is not None else None
        if type(timeout) not in (int, float) or not math.isfinite(timeout) or timeout <= 0:
            raise ValueError("timeout must be positive and finite")
        self._timeout = timeout

    def _request(self, method, path, params, *, secret=None, quote=False):
        url = self._quote if quote else self._base
        if url is None:
            raise ValueError("quote_url is required for quotes")
        if not quote:
            pairs = sorted(dict(params, secret_key=secret).items())
            raw = "&".join(f"{k}={v}" for k, v in pairs).encode()
            params = dict(params, sign=hashlib.md5(raw, usedforsecurity=False).hexdigest())
        payload = urlencode(params)
        target = url.path.rstrip("/") + path
        headers = {"Accept": "application/json"}
        body = None
        if method == "GET":
            target += "?" + payload
        else:
            headers["Content-Type"] = "application/x-www-form-urlencoded"
            body = payload.encode("ascii")
        transport = http.client.HTTPSConnection if url.scheme == "https" else http.client.HTTPConnection
        conn = transport(url.hostname, url.port, timeout=self._timeout)
        limit = 8 << 20
        try:
            conn.request(method, target, body=body, headers=headers)
            with conn.getresponse() as response:
                if not 200 <= response.status < 300:
                    raise HTTPError(response.status)
                data = response.read(limit + 1)
        except (OSError, http.client.HTTPException, UnicodeError):
            raise TransportError("BTCC request failed or timed out") from None
        finally:
            conn.close()
        if len(data) > limit:
            raise ProtocolError("response exceeds 8 MiB")
        try:
            result = json.loads(data, parse_float=Decimal, parse_constant=_number)
            if not isinstance(result, dict) or type(result.get("code")) is not int:
                raise ProtocolError("invalid or missing response code")
            code = result.pop("code")
            message = result.pop("msg", "")
            if code:
                raise APIError(code, message)
            return _amounts(result)
        except (ValueError, InvalidOperation, RecursionError):
            raise ProtocolError("invalid JSON or response fields") from None

    def login(self, *, user_name, password, api_key, secret_key, company_id=1) -> Session:
        params = {"user_name": _text(user_name, "user_name"), "password": _text(password, "password"),
                  "api_key": _text(api_key, "api_key"), "company_id": _uint(company_id, "company_id", bits=32) or 1}
        result = self._request("POST", "/v1/user/login", params, secret=_text(secret_key, "secret_key"))
        account = _field(result, "account")
        try:
            return Session(self, _text(result.get("token"), "token"), secret_key,
                           _uint(account.get("id"), "accountid", minimum=1),
                           _uint(account.get("account_groupid"), "group", minimum=1))
        except ValueError:
            raise ProtocolError("incomplete login response") from None

    def _call(self, session, method, path, *, group=False, **params):
        if not isinstance(session, Session) or session._client is not self:
            raise ValueError("session must belong to this client")
        params["token"] = session._token
        if group:
            params["account_group_id"] = session._group_id
        else:
            params["accountid"] = session._account_id
        return self._request(method, path, params, secret=session._secret)

    def keep_alive(self, session):
        return self._call(session, "GET", "/v1/user/keepalive")

    def get_symbol_list(self, session):
        return _field(self._call(session, "GET", "/v1/config/symbollist", group=True), "symbols", list)

    def get_account_group(self, session):
        return _field(self._call(session, "GET", "/v1/config/accountgroup", group=True), "account_group")

    def get_account(self, session):
        return _field(self._call(session, "GET", "/v1/account/account"), "account")

    def get_position_list(self, session, *, page_no=1, page_size=100, symbol=None, position_id=0):
        params = {"pageNo": _uint(page_no, "page_no") or 1, "pageSize": _uint(page_size, "page_size") or 100}
        if symbol is not None:
            params["symbol"] = _text(symbol, "symbol")
        if _uint(position_id, "position_id"):
            params["id"] = position_id
        result = self._call(session, "GET", "/v1/account/positionlist", **params)
        _field(result, "positions", list)
        if type(result.get("total")) is not int or result["total"] < 0:
            raise ProtocolError("invalid position count")
        return result

    def get_quote_price(self, symbol):
        quote = _field(self._request("GET", "/quot/reqOnePrdPrice", {"Symbol": _text(symbol, "symbol")}, quote=True), "data")
        if "Price" not in quote:
            raise ProtocolError("missing quote price")
        return quote

    @staticmethod
    def _order(symbol, direction, request_volume, request_price):
        if type(direction) is not int or direction not in (1, 2):
            raise ValueError("direction must be 1 (buy) or 2 (sell)")
        return {"symbol": _text(symbol, "symbol"), "direction": direction,
                "request_volume": _amount(request_volume, positive=True),
                "request_price": _amount(request_price, positive=True)}

    def open_position(self, session, *, symbol, direction, request_volume, request_price, multiple,
                      ref_id=0, ref_account_id=0, stop_loss=0, take_profit=0):
        params = self._order(symbol, direction, request_volume, request_price)
        params["multiple"] = _uint(multiple, "multiple", minimum=1, bits=16)
        for k, v in (("refid", ref_id), ("ref_accid", ref_account_id)):
            if _uint(v, k):
                params[k] = v
        for k, v in (("stop_loss", stop_loss), ("take_profit", take_profit)):
            amount = _amount(v)
            if amount != "0":
                params[k] = amount
        result = self._call(session, "POST", "/v1/account/openposition", **params)
        _field(result, "position")
        return result

    def close_position(self, session, *, position_id, symbol, direction, request_volume, request_price, ref_id=0):
        params = self._order(symbol, direction, request_volume, request_price)
        params["positionid"] = _uint(position_id, "position_id", minimum=1)
        if _uint(ref_id, "ref_id"):
            params["refid"] = ref_id
        return _field(self._call(session, "POST", "/v1/account/closeposition", **params), "position")

    def update_position(self, session, *, position_id, symbol, stop_loss, take_profit):
        return _field(self._call(session, "POST", "/v1/account/updateposition",
                                 id=_uint(position_id, "position_id", minimum=1), symbol=_text(symbol, "symbol"),
                                 stop_loss=_amount(stop_loss), take_profit=_amount(take_profit)), "position")
