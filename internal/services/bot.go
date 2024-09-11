package services

import (
	"bytes"
	"millionaire/internal/assets"
	"millionaire/internal/models"
	"os"
	"time"

	tele "gopkg.in/telebot.v3"

	initdata "github.com/telegram-mini-apps/init-data-golang"
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

func (bot *Bot) ValidateInitData(dataStr string) (*models.UserFromAuth, error) {
	// err := initdata.Validate(dataStr, bot.token, 0)
	// if err != nil {
	// 	return nil, err
	// }

	println("dataStr", dataStr)

	data, err := initdata.Parse(dataStr)
	if err != nil {
		return nil, err
	}

	return &models.UserFromAuth{
		ID:           data.User.ID,
		Username:     data.User.Username,
		FirstName:    data.User.FirstName,
		LastName:     data.User.LastName,
		IsBot:        data.User.IsBot,
		IsPremium:    data.User.IsPremium,
		LanguageCode: data.User.LanguageCode,
		PhotoURL:     data.User.PhotoURL,
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
