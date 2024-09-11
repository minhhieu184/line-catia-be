package main

import (
	"bytes"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"millionaire/internal/assets"
	"os"
)

func commandStart(c tele.Context) error {
	return c.Send(
		&tele.Photo{
			File:    tele.FromReader(bytes.NewReader(assets.BackgroundImage)),
			Caption: textStart,
		},
		&tele.SendOptions{
			ParseMode: tele.ModeHTML,
			ReplyMarkup: &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{
					{{Text: "🌙 Play Now", WebApp: &tele.WebApp{URL: os.Getenv("TELEGRAM_WEB_APP_URL")}}},
					{{Text: "🔊 Lastest news", URL: os.Getenv("TELEGRAM_ANNOUNCEMENT_URL")}},
					{{Text: "📱 Follow us on Twitter", URL: os.Getenv("TWITTER_URL")}},
				},
			},
		})
}

func commandHello(c tele.Context) error {
	return c.Send("Ready, Set, Go with the Catia quests!\nLet's embark on this adventure together")
}

func commandMe(c tele.Context) error {
	return c.Send(fmt.Sprintf("Hi %s. Let's play", c.Sender().Username))
}

func commandList(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	return c.Send(`List of commands:
/me - Get user info (public)
/notify all force - Send message to all users
/kol <username> <ref link> - Add ref link for KOL
/stats - Get total users
/kolstats <all/kol> [limit] - Get top invited KOLs
`)
}

func handlePrivacyCommands(b *tele.Bot) {
	menuPrivacy := &tele.ReplyMarkup{ResizeKeyboard: true}
	menuPrivacyWhatWeCollect := &tele.ReplyMarkup{ResizeKeyboard: true}
	menuPrivacyHowWeCollect := &tele.ReplyMarkup{ResizeKeyboard: true}
	menuPrivacyWhatWeDo := &tele.ReplyMarkup{ResizeKeyboard: true}
	menuPrivacyWhatWeNotDo := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnPrivacyWhatWeCollect := (&tele.ReplyMarkup{}).Data(textPrivacyWhatWeCollectBtn, "privacy-what-we-collect")
	btnHighlightPrivacyWhatWeCollect := (&tele.ReplyMarkup{}).Data(fmt.Sprintf(">> %s <<", textPrivacyWhatWeCollectBtn), btnPrivacyWhatWeCollect.Unique)
	btnPrivacyHowWeCollect := (&tele.ReplyMarkup{}).Data(textPrivacyHowWeCollectBtn, "privacy-how-we-collect")
	btnHighlightPrivacyHowWeCollect := (&tele.ReplyMarkup{}).Data(fmt.Sprintf(">> %s <<", textPrivacyHowWeCollectBtn), btnPrivacyHowWeCollect.Unique)
	btnPrivacyWhatWeDo := (&tele.ReplyMarkup{}).Data(textPrivacyWhatWeDoBtn, "privacy-what-we-do")
	btnHighlightPrivacyWhatWeDo := (&tele.ReplyMarkup{}).Data(fmt.Sprintf(">> %s <<", textPrivacyWhatWeDoBtn), btnPrivacyWhatWeDo.Unique)
	btnPrivacyWhatWeNotDo := (&tele.ReplyMarkup{}).Data(textPrivacyWhatWeNotDoBtn, "privacy-how-we-do")
	btnHighlightPrivacyWhatWeNotDo := (&tele.ReplyMarkup{}).Data(fmt.Sprintf(">> %s <<", textPrivacyWhatWeNotDoBtn), btnPrivacyWhatWeNotDo.Unique)

	btnBack := (&tele.ReplyMarkup{}).Data(textPrivacyBack, "privacy-back")
	username := ""
	if b.Me != nil {
		username = b.Me.Username
	}

	menuPrivacy.Inline(
		menuPrivacy.Row(btnPrivacyWhatWeCollect),
		menuPrivacy.Row(btnPrivacyHowWeCollect),
		menuPrivacy.Row(btnPrivacyWhatWeDo),
		menuPrivacy.Row(btnPrivacyWhatWeNotDo),
	)

	menuPrivacyWhatWeCollect.Inline(
		menuPrivacy.Row(btnHighlightPrivacyWhatWeCollect),
		menuPrivacy.Row(btnPrivacyHowWeCollect),
		menuPrivacy.Row(btnPrivacyWhatWeDo),
		menuPrivacy.Row(btnPrivacyWhatWeNotDo),
		menuPrivacyWhatWeCollect.Row(btnBack),
	)

	menuPrivacyHowWeCollect.Inline(
		menuPrivacy.Row(btnPrivacyWhatWeCollect),
		menuPrivacy.Row(btnHighlightPrivacyHowWeCollect),
		menuPrivacy.Row(btnPrivacyWhatWeDo),
		menuPrivacy.Row(btnPrivacyWhatWeNotDo),
		menuPrivacyHowWeCollect.Row(btnBack),
	)

	menuPrivacyWhatWeDo.Inline(
		menuPrivacy.Row(btnPrivacyWhatWeCollect),
		menuPrivacy.Row(btnPrivacyHowWeCollect),
		menuPrivacy.Row(btnHighlightPrivacyWhatWeDo),
		menuPrivacy.Row(btnPrivacyWhatWeNotDo),
		menuPrivacyWhatWeDo.Row(btnBack),
	)

	menuPrivacyWhatWeNotDo.Inline(
		menuPrivacy.Row(btnPrivacyWhatWeCollect),
		menuPrivacy.Row(btnPrivacyHowWeCollect),
		menuPrivacy.Row(btnPrivacyWhatWeDo),
		menuPrivacy.Row(btnHighlightPrivacyWhatWeNotDo),
		menuPrivacyWhatWeNotDo.Row(btnBack),
	)

	b.Handle(textCommandPrivacy, func(c tele.Context) error {
		return c.Send(fmt.Sprintf(textPrivacy, username), &tele.SendOptions{
			ReplyMarkup: menuPrivacy,
			ParseMode:   tele.ModeHTML,
		})
	})

	b.Handle(&btnBack, func(c tele.Context) error {
		return c.Send(fmt.Sprintf(textPrivacy, username), &tele.SendOptions{
			ReplyMarkup: menuPrivacy,
			ParseMode:   tele.ModeHTML,
		})
	})

	b.Handle(&btnPrivacyWhatWeCollect, func(c tele.Context) error {
		return c.Send(textPrivacyWhatWeCollect, &tele.SendOptions{
			ReplyMarkup: menuPrivacyWhatWeCollect,
			ParseMode:   tele.ModeHTML,
		})
	})

	b.Handle(&btnPrivacyHowWeCollect, func(c tele.Context) error {
		return c.Send(textPrivacyHowWeCollect, &tele.SendOptions{
			ReplyMarkup: menuPrivacyHowWeCollect,
			ParseMode:   tele.ModeHTML,
		})
	})

	b.Handle(&btnPrivacyWhatWeDo, func(c tele.Context) error {
		return c.Send(textPrivacyWhatWeDo, &tele.SendOptions{
			ReplyMarkup: menuPrivacyWhatWeDo,
			ParseMode:   tele.ModeHTML,
		})
	})

	b.Handle(&btnPrivacyWhatWeNotDo, func(c tele.Context) error {
		return c.Send(textPrivacyWhatWeNotDo, &tele.SendOptions{
			ReplyMarkup: menuPrivacyWhatWeNotDo,
			ParseMode:   tele.ModeHTML,
		})
	})
}
