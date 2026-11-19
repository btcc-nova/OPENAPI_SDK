package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	btcc "github.com/btcc-nova/OPENAPI_SDK"
)

func main() {
	client, err := btcc.NewClient(btcc.Options{
		BaseURL:  os.Getenv("BTCC_BASE_URL"),
		QuoteURL: os.Getenv("BTCC_QUOTE_URL"), // 不查询行情时可省略。
		Timeout:  10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.Login(ctx, btcc.LoginRequest{
		UserName:  os.Getenv("BTCC_USER_NAME"),
		Password:  os.Getenv("BTCC_PASSWORD"),
		APIKey:    os.Getenv("BTCC_API_KEY"),
		SecretKey: os.Getenv("BTCC_SECRET_KEY"),
		CompanyID: 1,
	})
	if err != nil {
		log.Fatal(err)
	}
	account, err := client.GetAccount(ctx, result.Session)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(account.ID, account.Balance.String())
}
