package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	http "github.com/Danny-Dasilva/fhttp"
	"github.com/joho/godotenv"
	"github.com/tonkeeper/tongo/boc"
	"github.com/tonkeeper/tongo/liteapi"
	"github.com/tonkeeper/tongo/tlb"
	"github.com/tonkeeper/tongo/ton"
	"github.com/tonkeeper/tongo/wallet"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

func init() {
	// for development
	//nolint:errcheck
	godotenv.Load("../../.env")

	// for production
	//nolint:errcheck
	godotenv.Load("./.env")
}

const (
	fragmentAPIUrl                    = "https://fragment.com/api?hash=%s"
	fragmentSearchUsernameStarsMethod = "searchStarsRecipient"
	fragmentInitBuyStarsRequestMethod = "initBuyStarsRequest"
	fragmentGetBuyStarsLinkMethod     = "getBuyStarsLink"
	ja3                               = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"
	userAgent                         = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36"
	cookie                            = ""
	apiHash                           = ""
	quantity                          = "50"
)

func main() {
	app := &cli.App{
		Name: "gift-stars-telegram",
		Commands: []*cli.Command{
			{
				Name:   "server",
				Action: action,
				Before: beforeAction,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func beforeAction(c *cli.Context) error {
	return nil
}

func action(c *cli.Context) error {
	transaction()

	return nil
}

func transaction() {
	client, err := liteapi.NewClientWithDefaultMainnet()
	w, err := wallet.DefaultWalletFromSeed(os.Getenv("PRIVATE_KEY_STARS"), client)
	if err != nil {
		fmt.Printf("Unable to create wallet: %v", err.Error())
		return
	}
	fmt.Printf("wallet %s\n", w.GetAddress().String())

	balance, err := w.GetBalance(context.Background())
	if err != nil {
		fmt.Println("GetBalance err:", err.Error())
		return
	}

	fmt.Printf("balance %d\n\n", balance)

	usernames := make([]string, 0)
	usernames = append(usernames)

	for _, u := range usernames {
		tx, err := sendTransaction(&w, u)
		if tx != "" && err != nil {
			fmt.Printf("User %s received as a gift in tx %s\n\n", u, tx)
			continue
		}

		if err != nil {
			fmt.Printf("transaction error: %s\n", err.Error())
			continue
		}

		fmt.Printf("User %s received as a gift in tx %s\n\n", u, tx)
	}
}

func sendTransaction(w *wallet.Wallet, username string) (string, error) {
	message, err := fragment(username)
	if err != nil {
		return "", fmt.Errorf("fragment error: %s", err.Error())
	}

	comment, err := base64.RawStdEncoding.DecodeString(message.Payload)
	if err != nil {
		return "", fmt.Errorf("decode comment error: %s", err.Error())
	}

	cells, err := boc.DeserializeBoc(comment)
	if err != nil {
		return "", fmt.Errorf("deserialize BOC error: %s", err.Error())
	}

	if len(cells) < 1 {
		return "", fmt.Errorf("cells error")
	}

	tx, err := w.SendV2(context.Background(), time.Second*15, wallet.Message{
		Amount:  tlb.Grams(message.Amount),
		Address: ton.MustParseAccountID(message.Address),
		Body:    cells[0],
		Code:    nil,
		Data:    nil,
		Bounce:  false,
		Mode:    3,
	})

	return tx.Hex(), err
}

func fragment(username string) (*BuyStarsLinkMessage, error) {
	payload := new(FragmentPayload)
	payload.Cookie = cookie
	payload.ApiHash = apiHash
	payload.Quantity = quantity

	recipient, err := getFragmentRecipient(payload, username)
	if err != nil {
		return nil, err
	}

	reqId, err := getFragmentRequestId(payload, recipient)
	if err != nil {
		return nil, err
	}

	message, err := getFragmentRefMessage(payload, reqId)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func doFragmentRequest(fragment *FragmentPayload, body io.Reader, response interface{}) error {
	if fragment == nil {
		return fmt.Errorf("fragment payload is empty")
	}

	cycleClient := &http.Client{
		Transport: cycletls.NewTransport(ja3, userAgent),
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(fragmentAPIUrl, fragment.ApiHash), body)
	if err != nil {
		return fmt.Errorf("create request error: %s", err.Error())
	}

	req.Header.Set("authority", "fragment.com")
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("cookie", fragment.Cookie)
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	res, err := cycleClient.Do(req)
	if err != nil {
		return fmt.Errorf("request error: %s", err.Error())
	}
	defer res.Body.Close()
	responseText, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response error: %s", err.Error())
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("request error with status code %d: %s", res.StatusCode, responseText)
	}

	if err := json.Unmarshal(responseText, response); err != nil {
		return fmt.Errorf("parse response error: %s", err.Error())
	}

	return nil
}

func getFragmentRequestId(fragment *FragmentPayload, recipient string) (string, error) {
	form := url.Values{}
	form.Add("recipient", recipient)
	form.Add("quantity", fragment.Quantity)
	form.Add("method", fragmentInitBuyStarsRequestMethod)

	response := new(InitBuyStarsResponse)
	err := doFragmentRequest(fragment, strings.NewReader(form.Encode()), response)
	if err != nil {
		return "", err
	}

	if response.Error != "" {
		return "", fmt.Errorf("get requestId error: %s", response.Error)
	}

	if response.ReqId == "" {
		return "", fmt.Errorf("requestId empty")
	}

	return response.ReqId, nil
}

func getFragmentRecipient(fragment *FragmentPayload, username string) (string, error) {
	form := url.Values{}
	form.Add("query", username)
	form.Add("quantity", fragment.Quantity)
	form.Add("method", fragmentSearchUsernameStarsMethod)

	response := new(StarsRecipientResponse)
	err := doFragmentRequest(fragment, strings.NewReader(form.Encode()), response)
	if err != nil {
		return "", err
	}

	if response.Error != "" {
		return "", fmt.Errorf("get recipient error: %s", response.Error)
	}

	if response.Found.Recipient == "" {
		return "", fmt.Errorf("recipient empty")
	}

	return response.Found.Recipient, nil
}

func getFragmentRefMessage(fragment *FragmentPayload, reqId string) (*BuyStarsLinkMessage, error) {
	form := url.Values{}
	form.Add("transaction", "1")
	form.Add("id", reqId)
	form.Add("show_sender", "0")
	form.Add("method", fragmentGetBuyStarsLinkMethod)

	response := new(GetBuyStarsLinkResponse)
	err := doFragmentRequest(fragment, strings.NewReader(form.Encode()), response)
	if err != nil {
		return nil, err
	}

	if response.Error != "" {
		return nil, fmt.Errorf("get refMessage error: %s", response.Error)
	}

	if !response.Ok || response.Transaction == nil || len(response.Transaction.Messages) < 1 {
		return nil, fmt.Errorf("refMessage empty")
	}

	return response.Transaction.Messages[0], nil
}
