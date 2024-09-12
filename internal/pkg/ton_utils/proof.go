package ton_utils

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tonkeeper/tongo"

	"encoding/base64"
	"encoding/binary"
)

const (
	tonProofPrefix   = "ton-proof-item-v2/"
	tonConnectPrefix = "ton-connect"
	expirationTime   = 24 * 60 * 60 // 1 hour
	siweValidV2_EXP  = 6 * time.Hour
)

func siweValidV2Key(address, nonce string) string {
	return fmt.Sprintf("nonce:%s:%s", address, nonce)
}

func SignatureVerify(pubkey ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(pubkey, message, signature)
}

func ParseTonProofMessage(tp *models.TonProof) (*models.TonProofMessage, error) {
	var TonProofMessage models.TonProofMessage

	addr, err := tongo.ParseAddress(tp.Address)
	if err != nil {
		return nil, err
	}
	sig, err := base64.StdEncoding.DecodeString(tp.Proof.Signature)
	if err != nil {
		return nil, err
	}

	TonProofMessage.Workchain = addr.ID.Workchain
	TonProofMessage.Address = addr.ID.Address[:]
	TonProofMessage.Domain = tp.Proof.Domain
	TonProofMessage.Timstamp = tp.Proof.Timestamp
	TonProofMessage.Signature = sig
	TonProofMessage.Payload = tp.Proof.Payload
	TonProofMessage.StateInit = tp.Proof.StateInit
	return &TonProofMessage, nil
}

func CreateMessage(message *models.TonProofMessage) ([]byte, error) {
	wc := make([]byte, 4)
	binary.BigEndian.PutUint32(wc, uint32(message.Workchain))

	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, uint64(message.Timstamp))

	dl := make([]byte, 4)
	binary.LittleEndian.PutUint32(dl, message.Domain.LengthBytes)
	m := []byte(tonProofPrefix)
	m = append(m, wc...)
	m = append(m, message.Address...)
	m = append(m, dl...)
	m = append(m, []byte(message.Domain.Value)...)
	m = append(m, ts...)
	m = append(m, []byte(message.Payload)...)
	messageHash := sha256.Sum256(m)
	fullMes := []byte{0xff, 0xff}
	fullMes = append(fullMes, []byte(tonConnectPrefix)...)
	fullMes = append(fullMes, messageHash[:]...)
	res := sha256.Sum256(fullMes)
	return res[:], nil
}

func CheckProof(ctx context.Context, dbRedis redis.UniversalClient, address tongo.AccountID, userID string, domain string, nonce string, tonProofReq *models.TonProofMessage) (bool, error) {
	if len(nonce) != 12 {
		return false, errors.New("invalid nonce")
	}

	if ok, err := CompareStateInitWithAddress(address, tonProofReq.StateInit); err != nil || !ok {
		return ok, err
	}

	pubKey, err := ParseStateInit(tonProofReq.StateInit)
	if err != nil {
		log.Printf("parse wallet state init error: %v\n", err)
		return false, err
	}

	if time.Now().After(time.Unix(tonProofReq.Timstamp, 0).Add(time.Duration(expirationTime) * time.Second)) {
		msgErr := "proof has been expired"
		log.Println(msgErr)
		return false, fmt.Errorf(msgErr)
	}

	key := siweValidV2Key(address.String(), nonce)
	n, err := redis_store.GetSIWTNonce(ctx, dbRedis, key)
	if err == nil && n != "" {
		return false, errors.New("used nonce")
	}

	err = redis_store.SetSIWTNonce(ctx, dbRedis, key, nonce, siweValidV2_EXP)
	if err != nil {
		return false, err
	}

	if tonProofReq.Domain.Value != domain {
		msgErr := fmt.Sprintf("wrong domain: %v", tonProofReq.Domain)
		log.Println(msgErr)
		return false, fmt.Errorf(msgErr)
	}

	mes, err := CreateMessage(tonProofReq)
	if err != nil {
		return false, err
	}

	return SignatureVerify(pubKey, mes, tonProofReq.Signature), nil
}
