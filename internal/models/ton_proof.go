package models

type TonDomain struct {
	LengthBytes uint32 `json:"lengthBytes"`
	Value       string `json:"value"`
}

type TonMessageInfo struct {
	Timestamp int64     `json:"timestamp"`
	Domain    TonDomain `json:"domain"`
	Signature string    `json:"signature"`
	Payload   string    `json:"payload"`
	StateInit string    `json:"state_init"`
}

type TonProof struct {
	Address string         `json:"address"`
	Nonce   string         `json:"nonce"`
	Proof   TonMessageInfo `json:"proof"`
}

type TonProofMessage struct {
	Workchain int32
	Address   []byte
	Timstamp  int64
	Domain    TonDomain
	Signature []byte
	Payload   string
	StateInit string
}
