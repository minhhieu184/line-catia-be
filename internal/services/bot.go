package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"millionaire/internal/assets"
	"millionaire/internal/models"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	tele "gopkg.in/telebot.v3"
)

const (
	textStart = `🌙 Welcome to Catia Eduverse!🌙

Explore quizzes, tasks & minigames and earn rewards.

 🚀 Join us in this exciting journey!

‼️ Tip: Pin Catia Eduverse app at the top of your Telegram for fastest access.
`
)

type Bot struct {
	token string
}

func NewBot(token string) (*Bot, error) {
	return &Bot{token}, nil
}

func (bot *Bot) Validate(idToken string) (*models.UserFromAuth, error) {
	// err := initdata.Validate(dataStr, bot.token, 0)
	// if err != nil {
	// 	return nil, err
	// }

	println("idToken", idToken)

	// decode idToken jwt
	data, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		return "6e61a64875ea78c130df3680d7991466", nil
	})
	if err != nil {
		println("err", err.Error())
		return nil, err
	}

	b, _ := json.MarshalIndent(data, "", "    ")
	fmt.Println("sdfjkbsdf")
	fmt.Println(string(b))

	return &models.UserFromAuth{
		// ID:           data.User.ID,
		// Username:     data.User.Username,
		// FirstName:    data.User.FirstName,
		// LastName:     data.User.LastName,
		// IsBot:        data.User.IsBot,
		// IsPremium:    data.User.IsPremium,
		// LanguageCode: data.User.LanguageCode,
		// PhotoURL:     data.User.PhotoURL,
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
