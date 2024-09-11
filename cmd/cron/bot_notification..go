package main

import (
	"context"
	"fmt"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/uptrace/bun"
	tele "gopkg.in/telebot.v3"
)

type BotNotificationJob struct {
	Redis     redis.UniversalClient
	Db        *bun.DB
	BotClient *tele.Bot
}

func NewBotNotificationJob(redis redis.UniversalClient, db *bun.DB, botClient *tele.Bot) *BotNotificationJob {
	return &BotNotificationJob{
		Redis:     redis,
		Db:        db,
		BotClient: botClient,
	}
}

func (c *BotNotificationJob) Start() {
	schedule := cron.New()
	timeline, err := datastore.GetConfigByKey(context.Background(), c.Db, "CRONJOB_TIME_SEND_MESSAGE")
	if err != nil {
		fmt.Println(err)
		return
	}

	if timeline == nil || timeline.Value == "" {
		fmt.Println("No timeline found")
		return
	}

	_, err = schedule.AddFunc(timeline.Value, c.scheduleSendMessage)
	fmt.Println("Bot Notification Cronjob start at:", time.Now().Format("2006-01-02 15:04:05"), "every:", timeline, err)
	c.scheduleSendMessage()
	schedule.Run()
}

func (c *BotNotificationJob) scheduleSendMessage() {
	fmt.Println("Start scan user countdown time ...")
	ctx := context.Background()

	//notifyDelayTimeStr := os.Getenv("NOTIFY_TIME")
	notifyDelayTimeStr, err := datastore.GetConfigByKey(ctx, c.Db, "NOTIFY_TIME")
	if err != nil {
		fmt.Println(err)
		return
	}
	notifyDelayTime, err := strconv.Atoi(notifyDelayTimeStr.Value)
	if err != nil {
		return
	}

	countdowns, err := datastore.GetCountdownCompletedUsers(ctx, c.Db, gameID, time.Now())
	if err != nil {
		return
	}

	fmt.Println("Number of users reached countdown time:", len(countdowns))

	msgConfig, _ := datastore.GetConfigByKey(ctx, c.Db, NOTIFY_AUTO_CONTENT)

	if msgConfig == nil || msgConfig.Value == "" {
		fmt.Println("No message config found")
		return
	}

	for _, user := range countdowns {
		lastNotify, err := redis_store.GetUserLastNotify(ctx, c.Redis, user.ID)
		if err == redis.Nil || time.Since(lastNotify) > time.Duration(notifyDelayTime)*time.Hour {
			// send message
			fmt.Println("User:", user.ID, "last notify time:", lastNotify, "sending message ...")
			_, err = c.BotClient.Send(tele.ChatID(user.ID), msgConfig.Value, &tele.SendOptions{
				ParseMode: tele.ModeHTML,
				ReplyMarkup: &tele.ReplyMarkup{
					InlineKeyboard: [][]tele.InlineButton{
						{{Text: "🎮 Play Now", WebApp: &tele.WebApp{URL: os.Getenv("TELEGRAM_WEB_APP_URL")}}},
						{{Text: "Join the community", URL: os.Getenv("TELEGRAM_COMMUNITY_URL")}},
					},
				},
			})

			if err == nil {
				// set to redis
				err = redis_store.SetUserLastNotify(ctx, c.Redis, user.ID, time.Now())
				if err != nil {
					fmt.Println("User:", user.ID, user.Username, "error set last notify time:", err)
				}
			} else {
				fmt.Println("User:", user.ID, user.Username, "error sending message:", err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("Done scan user countdown time ...")
}
