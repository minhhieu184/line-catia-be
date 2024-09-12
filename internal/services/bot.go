package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"millionaire/internal/assets"
	"millionaire/internal/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	tele "gopkg.in/telebot.v3"
)

const (
	textStart = `🌙 Welcome to Catia Eduverse!🌙

Explore quizzes, tasks & minigames and earn rewards.

 🚀 Join us in this exciting journey!

‼️ Tip: Pin Catia Eduverse app at the top of your Telegram for fastest access.
`
)

type LineVerifyResponse struct {
	Iss     string `json:"iss"`
	Sub     string `json:"sub"`
	Aud     string `json:"aud"`
	Exp     int64  `json:"exp"`
	Iat     int64  `json:"iat"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

type Bot struct {
	token string
}

func NewBot(token string) (*Bot, error) {
	return &Bot{token}, nil
}

func (bot *Bot) Validate(idToken string) (*models.UserFromAuth, error) {
	data := url.Values{}
	data.Set("id_token", idToken)
	data.Set("client_id", os.Getenv("LINE_CHANNEL_ID"))
	ioReader := strings.NewReader(data.Encode())
	res, err := http.Post(LINE_API_BASE_URL+"/verify", "application/x-www-form-urlencoded", ioReader)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		println("Line verify failed " + string(body))
		return nil, errorx.Wrap(errors.New("Line verify failed "+string(body)), errorx.Authn)
	}

	println("line verify", string(body))
	var lineVerifyResponse LineVerifyResponse
	if err := json.Unmarshal(body, &lineVerifyResponse); err != nil { // Parse []byte to go struct pointer
		return nil, err
	}
	b, _ := json.MarshalIndent(lineVerifyResponse, "", "    ")
	println("lineVerifyResponse", string(b))

	return &models.UserFromAuth{
		ID:       lineVerifyResponse.Sub,
		Username: lineVerifyResponse.Name,
		PhotoURL: lineVerifyResponse.Picture,
	}, nil
}

func (bot *Bot) SendMsg(chatID int64, text string) error {
	pref := tele.Settings{
		Token:  bot.token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	_, err = b.Send(&tele.User{ID: chatID}, text, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
		ReplyMarkup: &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{
				{{Text: "🌙 Play Now", WebApp: &tele.WebApp{URL: os.Getenv("TELEGRAM_WEB_APP_URL")}}},
				{{Text: "🔊 Lastest news", URL: os.Getenv("TELEGRAM_ANNOUNCEMENT_URL")}},
				{{Text: "📱 Follow us on Twitter", URL: os.Getenv("TWITTER_URL")}},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (bot *Bot) SendWelcomeMsg(chatID int64) error {
	pref := tele.Settings{
		Token:  bot.token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	_, err = b.Send(&tele.User{ID: chatID}, &tele.Photo{
		File:    tele.FromReader(bytes.NewReader(assets.BackgroundImage)),
		Caption: textStart,
	}, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
		ReplyMarkup: &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{
				{{Text: "🌙 Play Now", WebApp: &tele.WebApp{URL: os.Getenv("TELEGRAM_WEB_APP_URL")}}},
				{{Text: "🔊 Lastest news", URL: os.Getenv("TELEGRAM_ANNOUNCEMENT_URL")}},
				{{Text: "📱 Follow us on Twitter", URL: os.Getenv("TWITTER_URL")}},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
