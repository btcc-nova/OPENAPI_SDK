import hashlib
import json
import os
from decimal import Decimal
from functools import partial
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import subprocess
import sys
from threading import Thread
import time
import unittest
from urllib.parse import parse_qsl, urlsplit

from btcc_sdk import APIError, Client, HTTPError, ProtocolError, TransportError

CREDENTIALS = dict(user_name="sdk+test@example.invalid", password="test&password",
                   api_key="test-key", secret_key="test-secret")
TOKEN = "fixture+&token"
ORDER = dict(symbol="A/B TC&", request_volume=Decimal("0.0100"),
             request_price="123.456789012345678901", direction=1)
ROUTES = {
    "/v1/user/login": {"token": TOKEN, "account": {"id": 21, "account_groupid": 7}},
    "/v1/user/keepalive": {"accountid": 21},
    "/v1/config/symbollist": {"symbols": [{"enable": 3, "volumes_min": "0.01"}]},
    "/v1/config/accountgroup": {"account_group": {"id": 7}},
    "/v1/account/account": {"account": {"id": 21, "balance": "123.456789012345678901"}},
    "/v1/account/positionlist": {"positions": [{"id": 3}], "total": 1},
    "/v1/account/openposition": {"position": {"id": 3}, "sltp_orders": []},
    "/v1/account/closeposition": {"position": {"id": 3}},
    "/v1/account/updateposition": {"position": {"id": 3}},
    "/quot/reqOnePrdPrice": {"data": {"Price": "1.2300", "Symbol": ORDER["symbol"]}},
}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self.reply(urlsplit(self.path).query)

    def do_POST(self):
        self.reply(self.rfile.read(int(self.headers['Content-Length'])).decode())

    def reply(self, encoded):
        parts = urlsplit(self.path)
        params = dict(parse_qsl(encoded, keep_blank_values=True))
        self.server.calls.append((self.command, parts.path, params, self.headers, parts.query))
        if self.server.delay:
            time.sleep(self.server.delay)
        if self.server.override is not None:
            status, body = self.server.override
        else:
            status, body = 200, json.dumps(dict(ROUTES[parts.path], code=0)).encode()
        self.send_response(status)
        if status == 302:
            self.send_header('Location', '/unexpected')
        self.end_headers()
        try:
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionResetError):
            pass


class ClientTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        def start():
            server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
            Thread(target=partial(server.serve_forever, poll_interval=0.02), daemon=True).start()
            cls.addClassCleanup(server.server_close)
            cls.addClassCleanup(server.shutdown)
            return server
        cls.trade, cls.quote = start(), start()
        cls.base = f'http://127.0.0.1:{cls.trade.server_port}'
        cls.quotes = f'http://127.0.0.1:{cls.quote.server_port}'

    def setUp(self):
        for server in (self.trade, self.quote):
            server.calls, server.override, server.delay = [], None, 0
        self.c = Client(self.base, quote_url=self.quotes)

    def login(self):
        return self.c.login(**CREDENTIALS)

    def test_all_ten_wire_contracts(self):
        s = self.login()
        method, path, params, *_ = self.trade.calls[-1]
        self.assertEqual(params, {"user_name": CREDENTIALS['user_name'], "password": 'test&password',
                                  "company_id": '1', "api_key": 'test-key',
                                  "sign": 'e8bbaf9c2d1f3d22221f49008acbf506'})  # Go vector
        self.assertNotIn(TOKEN, repr(s))
        self.assertEqual(self.c.keep_alive(s), {"accountid": 21})
        self.assertEqual(self.c.get_symbol_list(s), [{"enable": 3, "volumes_min": Decimal('.01')}])
        self.assertEqual(self.c.get_account_group(s)['id'], 7)
        self.assertEqual(self.c.get_account(s)['balance'], Decimal(ORDER['request_price']))
        self.assertEqual(self.c.get_position_list(s, page_no=2, page_size=4, symbol='A/B', position_id=3),
                         {'total': 1, 'positions': [{'id': 3}]})
        self.assertEqual(self.c.open_position(s, **ORDER, multiple=2, ref_id=8, ref_account_id=9,
                                              take_profit='160')['position']['id'], 3)
        self.assertEqual(self.c.update_position(s, position_id=3, symbol='A/B', stop_loss=0, take_profit='160'), {'id': 3})
        self.assertEqual(self.c.close_position(s, **ORDER, position_id=3, ref_id=10), {'id': 3})
        self.assertEqual(self.c.get_quote_price(ORDER['symbol'])['Price'], Decimal('1.23'))
        self.assertEqual(len(self.trade.calls), 9)
        self.assertEqual(len(self.quote.calls), 1)
        q_method, q_path, q_params, q_headers, _ = self.quote.calls[0]
        self.assertEqual((q_method, q_path, q_params), ('GET', '/quot/reqOnePrdPrice', {'Symbol': ORDER['symbol']}))
        self.assertNotIn('Authorization', q_headers)
        self.assertNotIn('Cookie', q_headers)
        calls = {row[1]: row[2] for row in self.trade.calls}
        self.assertEqual(set(calls) | {q_path}, set(ROUTES))
        for method, path, params, headers, query in self.trade.calls:
            post = path.endswith(('login', 'openposition', 'closeposition', 'updateposition'))
            self.assertEqual(method, 'POST' if post else 'GET')
            if post:
                self.assertEqual(query, '')
                self.assertEqual(headers.get_content_type(), 'application/x-www-form-urlencoded')
            self.assertNotIn('secret_key', params)
            if path.endswith('login'):
                continue
            self.assertEqual(params['token'], TOKEN)
            self.assertNotIn('api_key', params)
            self.assertNotIn('password', params)
            if '/config/' in path:
                self.assertEqual(params['account_group_id'], '7')
                self.assertNotIn('accountid', params)
            else:
                self.assertEqual(params['accountid'], '21')
                self.assertNotIn('account_group_id', params)
            signed = dict(params, secret_key=CREDENTIALS['secret_key'])
            signature = signed.pop('sign')
            canonical = '&'.join(f'{k}={signed[k]}' for k in sorted(signed))
            self.assertEqual(signature, hashlib.md5(canonical.encode()).hexdigest())
        opened = calls['/v1/account/openposition']
        for k, v in dict(direction='1', request_volume='0.01', request_price=ORDER['request_price'],
                         symbol=ORDER['symbol'], refid='8', ref_accid='9', multiple='2', take_profit='160').items():
            self.assertEqual(opened[k], v)
        self.assertNotIn('stop_loss', opened)
        page = calls['/v1/account/positionlist']
        self.assertEqual([page[k] for k in ('pageNo', 'pageSize', 'symbol', 'id')], ['2', '4', 'A/B', '3'])
        update, close = calls['/v1/account/updateposition'], calls['/v1/account/closeposition']
        self.assertEqual([update[k] for k in ('id', 'stop_loss', 'take_profit')], ['3', '0', '160'])
        self.assertEqual([close[k] for k in ('positionid', 'direction', 'refid')], ['3', '1', '10'])

    def test_precision_and_defaults(self):
        s = self.login()
        self.c.get_position_list(s)
        page = self.trade.calls[-1][2]
        self.assertEqual((page['pageNo'], page['pageSize']), ('1', '100'))
        self.quote.override = (200, b'{"code":0,"data":{"Price":123456.1234567890123456789}}')
        self.assertEqual(self.c.get_quote_price('X')['Price'], Decimal('123456.1234567890123456789'))

    def test_api_and_http_errors_never_retry_or_follow(self):
        s = self.login()
        for status, body, error, code in [(200, b'{"code":4,"msg":"fixture+&token"}', APIError, 4),
                                         (302, b'secret body', HTTPError, 302), (503, b'', HTTPError, 503)]:
            with self.subTest(status=status):
                self.trade.override = (status, body)
                before = len(self.trade.calls)
                with self.assertRaises(error) as exc:
                    self.c.close_position(s, position_id=3, **ORDER)
                self.assertEqual(len(self.trade.calls), before + 1)
                e = exc.exception
                self.assertEqual(e.code if isinstance(e, APIError) else e.status_code, code)
                self.assertNotIn(TOKEN, str(e))
                if isinstance(e, APIError):
                    self.assertEqual(e.message, TOKEN)
        self.quote.override = (200, b'{"code":-4,"msg":"no quote"}')
        with self.assertRaises(APIError) as exc:
            self.c.get_quote_price('X')
        self.assertEqual(exc.exception.code, -4)

    def test_input_rejected_before_network(self):
        s = self.login()
        for field, invalid in [('request_price', float('nan')), ('request_volume', .01), ('request_volume', '0'),
                               ('request_price', '-1'), ('request_price', 'Infinity'), ('symbol', ' '),
                               ('direction', True), ('direction', 3), ('multiple', 65536)]:
            with self.subTest(field=field, invalid=invalid), self.assertRaises(ValueError):
                self.c.open_position(s, **{**ORDER, "multiple": 1, field: invalid})
        with self.assertRaises(ValueError):
            Client(self.base).get_account(s)
        with self.assertRaises(ValueError):
            self.c.get_position_list(s, page_no=-1)
        with self.assertRaises(ValueError):
            self.c.update_position(s, position_id=0, symbol='X', stop_loss=0, take_profit=0)
        with self.assertRaises(ValueError):
            Client(self.base).get_quote_price('X')
        self.assertEqual(len(self.trade.calls), 1)
        for url in ('ftp://example.invalid', 'https://user@host', 'http://host:65536', 'https://host?x', 'https://host/#secret'):
            with self.subTest(url=url), self.assertRaises(ValueError):
                Client(url)

    def test_protocol_and_transport_failures(self):
        for body in [b'bad', b'null', b'{"code":false}', b'{}', b'{"code":0}', b'{"code":0,"data":null}',
                     b'{"code":0,"data":{"Price":"nan"}}', b'x' * ((8 << 20) + 1)]:
            self.quote.override = (200, body)
            with self.assertRaises(ProtocolError):
                self.c.get_quote_price('X')
        self.trade.override = (200, b'{"code":0,"account":{"id":1}}')
        with self.assertRaises(ProtocolError):
            self.login()
        self.quote.delay = .1
        with self.assertRaises(TransportError):
            Client(self.base, quote_url=self.quotes, timeout=.02).get_quote_price('X')

    def test_read_only_example(self):
        env = dict(os.environ, BTCC_BASE_URL=self.base, BTCC_USER_NAME=CREDENTIALS['user_name'],
                   BTCC_PASSWORD=CREDENTIALS['password'], BTCC_API_KEY=CREDENTIALS['api_key'],
                   BTCC_SECRET_KEY=CREDENTIALS['secret_key'], BTCC_COMPANY_ID='1')
        example = Path(__file__).resolve().parents[1] / 'examples' / 'login.py'
        output = subprocess.check_output([sys.executable, str(example)], env=env, text=True, timeout=10)
        self.assertEqual(output.strip(), 'account=21 balance=123.456789012345678901')
        self.assertEqual([row[1] for row in self.trade.calls], ['/v1/user/login', '/v1/account/account'])
