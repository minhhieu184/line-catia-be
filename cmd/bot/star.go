package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
	"millionaire/internal/datastore/redis_store"
	"strconv"
	"strings"
)

func handleStarCommands(b *tele.Bot) {
	b.Handle("/donate", func(c tele.Context) error {
		if !AuthRequire(c, chatId) {
			return nil
		}

		dbRedis, err := getContextRedis(c)
		if err != nil {
			return err
		}

		invoiceId := uuid.New().String()
		invoice := &tele.Invoice{
			Title:       "Donation",
			Description: "Donate Stars to Catia",
			Payload:     invoiceId,
			Currency:    "XTR",
			Prices: []tele.Price{
				{
					Label:  "Star",
					Amount: 5,
				},
			},
		}

		message, err := invoice.Send(b, c.Recipient(), nil)
		if err != nil {
			return c.Send(fmt.Errorf("create invoice error %s", err.Error()))
		}

		if message == nil {
			return c.Send("create invoice message error")
		}

		redis_store.SetInvoiceMessage(context.Background(), dbRedis, invoiceId, &tele.StoredMessage{
			MessageID: strconv.Itoa(message.ID),
			ChatID:    message.Chat.ID,
		})

		return nil
	})

	b.Handle(tele.OnCheckout, func(c tele.Context) error {
		dbRedis, err := getContextRedis(c)
		if err != nil {
			return err
		}

		query := c.PreCheckoutQuery()
		invoiceMessage, err := redis_store.GetInvoiceMessage(context.Background(), dbRedis, query.Payload)
		if err != nil {
			return err
		}

		if err := b.Accept(query); err != nil {
			return c.Send(fmt.Sprintf("donate error %s", err.Error()))
		}

		user := c.Sender()
		if user == nil {
			return fmt.Errorf("invalid user")
		}

		username := fmt.Sprintf("@%s", user.Username)
		if c.Sender().Username == "" {
			username = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		}
		username = strings.TrimSpace(username)

		b.Send(tele.ChatID(invoiceMessage.ChatID),
			fmt.Sprintf("%s thank you for your donation! You’re helping us continue our mission in a meaningful way.", username))
		b.Delete(invoiceMessage)

		return nil
	})
}
