package main

type StarsRecipientResponse struct {
	OK    bool           `json:"ok"`
	Found StarsRecipient `json:"found"`
	Error string         `json:"error"`
}

type StarsRecipient struct {
	MySelf    bool   `json:"myself"`
	Name      string `json:"name"`
	Photo     string `json:"photo"`
	Recipient string `json:"recipient"`
}

type InitBuyStarsResponse struct {
	Amount  string `json:"amount"`
	Button  string `json:"button"`
	Content string `json:"content"`
	MySelf  bool   `json:"myself"`
	ReqId   string `json:"req_id"`
	Error   string `json:"error"`
}

type BuyStarsLinkMessage struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
	Payload string `json:"payload"`
}

type GetBuyStarsLinkResponse struct {
	ConfirmMethod string `json:"confirm_method"`
	ConfirmParams struct {
		Id string `json:"id"`
	} `json:"confirm_params"`
	Ok          bool `json:"ok"`
	Transaction *struct {
		From       string                 `json:"from"`
		ValidUntil int64                  `json:"validUntil"`
		Messages   []*BuyStarsLinkMessage `json:"messages"`
	} `json:"transaction"`
	Error string `json:"error"`
}

type FragmentPayload struct {
	ApiHash  string
	Cookie   string
	Quantity string
}
