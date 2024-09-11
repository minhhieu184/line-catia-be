package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"millionaire/internal/pkg/caching"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/hiendaovinh/toolkit/pkg/env"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/urfave/cli/v2"
	tele "gopkg.in/telebot.v3"
)

func init() {
	// for development
	//nolint:errcheck
	godotenv.Load("../../.env")

	// for production
	//nolint:errcheck
	godotenv.Load("./.env")
}

type LinkCount struct {
	Description string
	Count       int
}

var chatId []int64

const (
	STATUS_BLOCKED        = "blocked"
	STATUS_CHAT_NOT_FOUND = "chat_not_found"
	STATUS_DEACTIVATED    = "deactivated"
)

const (
	textStart = `🌙 Welcome to Catia Eduverse!🌙

Explore quizzes, tasks & minigames and earn rewards.

 🚀 Join us in this exciting journey!

‼️ Tip: Pin Catia Eduverse app at the top of your Telegram for fastest access.
`
	textPrivacy = `<b>PRIVACY POLICY</b>
<i>Last updated on July 5, 2024</i>

<b>Mini app</b>: Catia Eduverse
<b>Telegram</b>: https://t.me/%s/app

Catia Eduverse aims to provide an engaging educational platform for learning about blockchain and top projects while preserving users’ privacy as best as possible.
We will review and may update this Privacy Policy from time to time. Please frequently check the last updated date to see any changes to our Privacy Policy.
Our privacy policy strictly adheres to all privacy laws and regulations of Telegram and doesn’t conflict, amend, or exhibit any discrepancies with the stipulations.`
	textPrivacyWhatWeCollect = `<b>The information we gather and handle at the moment includes:</b>

• Your public Telegram details, such as User ID, first name, last name, and username
• The input information in the app (optional), such as your wallet address and social media accounts connected to the app other than Telegram
• We may request that you receive push notifications regarding your account or specific features`

	textPrivacyHowWeCollect = `<b>We mainly process the information that you directly provide for specific purposes:</b>

• You agree to access the app via commands or access the direct links to the app
• You proactively input your information into the app`
	textPrivacyWhatWeDo = `<b>We use your public and input information to support various app experiences and features. This can include:</b>

• User ID/username pairing, which allows the app to resolve usernames to valid user IDs and verify whether you are a bot or not
• Giving membership keys to private Telegram groups for additional app support or access to exclusive app features and content.
• Reading particular messages sent by users`
	textPrivacyWhatWeNotDo = `<b>We do not:</b>

• Use technologies like beacons or unique device identifiers to identify you or your device
• Knowingly contact or collect personal information from children under 13. If you believe we have inadvertently collected such information, please get in touch with us so we can promptly obtain parental consent or remove the information
• Share any sensitive information with any other organisations or individuals`

	textPrivacyWhatWeCollectBtn = "What information we collects and stores"
	textPrivacyHowWeCollectBtn  = "How we collect your information"
	textPrivacyWhatWeDoBtn      = "What we do with your information"
	textPrivacyWhatWeNotDoBtn   = "What we do not do with your information"
	textPrivacyBack             = "⬅ Back"
	textCommandPrivacy          = "/privacy"

	contextRedis      = "context-redis"
	contextRedisCache = "context-redis-cache"
	contextPostgres   = "context-postgres"
)

func main() {
	app := &cli.App{
		Name: "bot-telegram",
		Commands: []*cli.Command{
			commandBot(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandBot() *cli.Command {
	return &cli.Command{
		Name:   "server",
		Action: action,
		Before: beforeAction,
	}
}

func beforeAction(c *cli.Context) error {
	return nil
}

func action(c *cli.Context) error {
	vs, err := env.EnvsRequired(
		"BOT_TOKEN",
	)
	if err != nil {
		return err
	}

	var dbRedis redis.UniversalClient
	var dbRedisCache redis.UniversalClient

	clusterRedisQuestionnaire := os.Getenv("CLUSTER_REDIS_QUESTIONNAIRE")
	if clusterRedisQuestionnaire != "" {
		clusterOpts, err := redis.ParseClusterURL(clusterRedisQuestionnaire)
		if err != nil {
			return err
		}
		dbRedis = redis.NewClusterClient(clusterOpts)
	} else {
		dbRedis, err = db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_QUESTIONNAIRE"),
		})
		if err != nil {
			return err
		}
	}

	clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_CACHE")
	if clusterCacheRedisURL != "" {
		clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
		if err != nil {
			return err
		}
		dbRedisCache = redis.NewClusterClient(clusterOpts)
	} else {
		dbRedisCache, err = db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_CACHE"),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	postgresDb, err := getDb()
	if err != nil {
		return err
	}

	chatIds, _ := datastore.GetConfigByKey(context.Background(), postgresDb, "ADMIN_CHAT_ID")

	if chatIds != nil {
		chatIdStrings := strings.Split(chatIds.Value, ",")

		for _, v := range chatIdStrings {
			vInt, _ := strconv.ParseInt(v, 10, 64)
			chatId = append(chatId, vInt)
		}
	}

	pref := tele.Settings{
		Token:  vs["BOT_TOKEN"],
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return err
	}

	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Callback() != nil {
				defer c.Respond()
			}

			c.Set(contextPostgres, postgresDb)
			c.Set(contextRedis, dbRedis)
			c.Set(contextRedisCache, dbRedisCache)
			//c.Set(contextCache, cache)

			return next(c)
		}
	})

	// static commands
	b.Handle("/start", commandStart)
	handlePrivacyCommands(b)
	b.Handle("/hello", commandHello)
	b.Handle("/me", commandMe)
	b.Handle("/list", commandList)

	//query commands
	b.Handle("/ref", commandCheckRef)
	b.Handle("/topref", commandCheckTop10Ref)
	b.Handle("/check_admin", commandCheckAdminOfChannel)
	// Stars commands
	handleStarCommands(b)

	// sub news
	b.Handle(tele.OnChannelPost, updateNews)

	// heavy commands - connect to database
	b.Handle("/kol", commandKol)
	b.Handle("/stats", commandStats)
	b.Handle("/kolstats", commandKOLStats)
	b.Handle("/setwallet", commandSetWallet)
	b.Handle("/notify", func(c tele.Context) error {
		if !AuthRequire(c, chatId) {
			return nil
		}

		postgresDb, err := getContextPostgres(c)
		if err != nil {
			return c.Send(fmt.Sprintf("error %s", err.Error()))
		}

		query := c.Args()
		if len(query) < 2 {
			return c.Send("Please enter the message you want to send!")
		}

		if query[0] != "all" {
			return c.Send("Invalid command")
		}

		if query[1] != "force" {
			return c.Send("Invalid command")
		}

		bannerFileName := ""
		if len(query) >= 3 {
			bannerFileName = query[2]
		}

		if bannerFileName == "" {
			bannerFileName = "bot_banner.jpg"
		}

		msg, err := datastore.GetConfigByKey(context.Background(), postgresDb, "NOTIFY_ALL_CONTENT")
		if err != nil {
			return c.Send("Error when get config: " + err.Error())
		}

		currentOffset := 0
		limit := 20
		image := tele.FromDisk(bannerFileName)
		for {
			users, err := datastore.GetChatAvailabledUsersSortedByCreatedAt(context.Background(), postgresDb, limit, currentOffset)
			if err != nil {
				return c.Send("Error when get users: " + err.Error())
			}

			if len(users) == 0 {
				break
			}

			waitgroup := sync.WaitGroup{}

			start := time.Now()

			for _, user := range users {
				sent, err := redis_store.GetSendMessageUser(context.Background(), dbRedis, user.ID)
				if err != nil && err != redis.Nil {
					c.Send("Error when get sent message: " + err.Error())
				}

				if sent {
					continue
				}

				waitgroup.Add(1)

				go func(user *models.User) {
					defer waitgroup.Done()

					if user == nil {
						return
					}
					u := tele.User{ID: user.ID}
					_, err = b.Send(&u, &tele.Photo{
						File:    image,
						Caption: msg.Value,
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
						msgError := fmt.Sprintf("Error when send message to user %d: %s", user.ID, err.Error())
						if user.Username != "" {
							msgError = fmt.Sprintf("Error when send message to username %s - userId: %d: %s", user.Username, user.ID, err.Error())
						}
						fmt.Println(msgError)
						switch err {
						case tele.ErrBlockedByUser:
							err = datastore.UpdateUserStatus(context.Background(), postgresDb, user.ID, STATUS_BLOCKED)
							if err != nil {
								c.Send("Error when update user status: " + err.Error())
							}
						case tele.ErrChatNotFound:
							err = datastore.UpdateUserStatus(context.Background(), postgresDb, user.ID, STATUS_CHAT_NOT_FOUND)
							if err != nil {
								c.Send("Error when update user status: " + err.Error())
							}
						case tele.ErrUserIsDeactivated:
							err = datastore.UpdateUserStatus(context.Background(), postgresDb, user.ID, STATUS_DEACTIVATED)
							if err != nil {
								c.Send("Error when update user status: " + err.Error())
							}
						default:
							c.Send("Error when send message: " + err.Error())
						}
					}

					err = redis_store.SetSendMessageUser(context.Background(), dbRedis, user.ID)
					if err != nil {
						c.Send("Error when set sent message: " + err.Error())
					}
				}(user)
			}
			waitgroup.Wait()

			currentOffset += limit

			fmt.Println("Send message to users: ", currentOffset)

			if time.Since(start) < 1*time.Second {
				time.Sleep(1 * time.Second)
			}

			time.Sleep(1 * time.Second)
		}

		fmt.Println("Send message to all users successfully")
		return c.Send("Send message to all users successfully")
	})

	b.Handle("/clear_my_social_tasks", func(c tele.Context) error {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		testers, _ := datastore.GetConfigByKey(ctx, postgresDb, "TESTERS")
		if testers == nil {
			return c.Send("Testers list not found")
		}

		testerIdStrings := strings.Split(testers.Value, ",")
		var testerIds []int64
		for _, v := range testerIdStrings {
			vInt, _ := strconv.ParseInt(v, 10, 64)
			testerIds = append(testerIds, vInt)
		}

		if !AuthRequireUsers(c, testerIds) {
			return nil
		}

		userID := c.Sender().ID
		user, err := datastore.FindUserByID(ctx, postgresDb, userID)

		if err != nil {
			return err
		}

		if user != nil {
			fmt.Println("User found, deleting...")
			caching.DeleteKeys(ctx, dbRedis, fmt.Sprintf("user:joined_social_link:%d:*", user.ID))
			caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("user:%d*", user.ID))
			caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("me:%d*", user.ID))
			caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("social_task:user:%d*", user.ID))
		}

		return c.Send(fmt.Sprintf("Your social tasks have been cleared successfully, %s", c.Sender().Username))
	})

	b.Handle("/list_task", commandGetAllTaskSlug)
	b.Handle("/task", commandGetTaskCount)

	b.Start()

	return nil
}

func getDb() (*bun.DB, error) {
	godotenv.Load()
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(os.Getenv("DB_DSN")),
		pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
	))

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}

func AuthRequire(ctx tele.Context, chatId []int64) bool {
	//if ctx.Message().Chat.ID != chatId {
	//	ctx.Send("You are not authorized to use this bot here.")
	//	return false
	//}
	//
	//return true

	authorized := false
	for _, id := range chatId {
		if ctx.Message().Chat.ID == id {
			authorized = true
			break
		}
	}

	if !authorized {
		ctx.Send("You are not authorized to use this bot here.")
	}

	return authorized
}

func AuthRequireUsers(ctx tele.Context, userIds []int64) bool {
	authorized := false
	for _, userId := range userIds {
		if ctx.Sender().ID == userId {
			authorized = true
			break
		}
	}

	if !authorized {
		ctx.Send("You are not authorized to use this bot here.")
	}

	return authorized
}

func commandKol(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	query := c.Args()
	if len(query) == 0 || len(query) < 2 {
		return c.Send("Please enter the KOL username and ref link you want to add!")
	}

	ctx := context.Background()

	username := query[0]
	refLink := query[1]

	flag, err := datastore.CheckRefCodeExists(ctx, postgresDb, refLink)
	if err != nil && err != sql.ErrNoRows {
		return c.Send("Error when check ref code: " + err.Error())
	}

	if flag {
		return c.Send("Ref code already exists")
	}

	user, _ := datastore.FindUserByUsername(ctx, postgresDb, username)
	if user != nil {
		user.RefCode = &refLink
		user.UpdatedAt = time.Now()
		_, err := datastore.EditUser(ctx, postgresDb, user)
		if err != nil {
			return c.Send("Error when update user: " + err.Error())
		}

		// TODO delete key user in cache
		// err = redis_store.DeleteUserRedis(ctx, dbRedis, user.ID)
		if err != nil {
			return c.Send("Error when update user: " + err.Error())
		}

		return c.Send(fmt.Sprintf("User %s has been updated successfully with %s as ref code", username, refLink))
	}

	return c.Send("Error: username not found")
}

func commandStats(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	count, err := datastore.CountUsers(context.Background(), postgresDb)
	if err != nil {
		return err
	}

	return c.Send(fmt.Sprintf("Total users: %d", count))
}

func commandKOLStats(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	query := c.Args()
	if len(query) < 1 {
		return c.Send("Please enter the Mode you want to get stats and number you want to query!")
	}

	mode := query[0]
	var limit string
	if len(query) > 1 {
		limit = query[1]
	} else {
		limit = "10"
	}
	//convert limit to nt
	limitInt, _ := strconv.Atoi(limit)
	if mode == "all" {
		kols, err := datastore.GetTopInvitedUsers(context.Background(), postgresDb, false, limitInt)
		if err != nil {
			return c.Send("Error when get top invited KOLs: " + err.Error())
		}

		msg := "Top invited KOLs:"
		for index, kol := range kols {
			msg += fmt.Sprintf("\n%d. %s - Invited: %d", index+1, kol.Username, kol.TotalInvites)
			if kol.RefCode != nil {
				msg += fmt.Sprintf(" (%s)", *kol.RefCode)
			}
		}
		return c.Send(msg)
	}

	kols, err := datastore.GetTopInvitedUsers(context.Background(), postgresDb, true, limitInt)
	if err != nil {
		return c.Send("Error when get top invited KOLs: " + err.Error())
	}

	msg := "Top invited KOLs:"
	for index, kol := range kols {
		msg += fmt.Sprintf("\n%d. %s - Invited: %d", index+1, kol.Username, kol.TotalInvites)
		if kol.RefCode != nil {
			msg += fmt.Sprintf(" (%s)", *kol.RefCode)
		}
	}
	return c.Send(msg)
}

func commandSetWallet(c tele.Context) error {
	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	ctx := context.Background()
	top50Str, err := datastore.GetConfigByKey(ctx, postgresDb, "TOP_50")
	if err != nil {
		return c.Send("Error when get config top 50: " + err.Error())
	}

	top50 := strings.Split(top50Str.Value, ",")
	var top50Int []int64
	for _, v := range top50 {
		vInt, _ := strconv.ParseInt(v, 10, 64)
		top50Int = append(top50Int, vInt)
	}

	if !AuthRequireUsers(c, top50Int) {
		return nil
	}

	query := c.Args()
	if len(query) < 1 {
		return c.Send("Invalid syntax. Please resend a new command following instruction below:\n/setwallet your_wallet_address")
	}

	walletAddress := strings.ToLower(query[0])

	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

	if !re.MatchString(walletAddress) {
		return c.Send("Invalid EVM wallet address format.")
	}

	userID := c.Sender().ID
	userWallet, _ := datastore.FindUserWalletByUserID(ctx, postgresDb, c.Sender().ID)
	if userWallet != nil && userWallet.EVMWallet != nil {
		evmWallet := *userWallet.EVMWallet
		return c.Send(fmt.Sprintf("You already have a wallet address ends with: *%s. \nIf you want to change it, please contact the admin for support.", evmWallet[len(evmWallet)-6:]))
	}

	userWallet, _ = datastore.FindUserWalletByEVMWallet(ctx, postgresDb, walletAddress)
	if userWallet != nil {
		return c.Send("Wallet address already exists in the system. \nPlease use another wallet address or contact the admin for support.")
	}

	uWallet := &models.UserWallet{
		ID:        userID,
		EVMWallet: &walletAddress,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = datastore.CreateUserWallet(ctx, postgresDb, uWallet)
	if err != nil {
		return c.Send("Something went wrong. \nPlease try again later or contact the admin for support.")
	}

	return c.Send("Your wallet is added successfully. Notice that you cannot change it. \nCatia Fan NFT will be airdropped to you soon. Thanks!")
}

func updateNews(c tele.Context) error {
	msg := c.Message()
	if msg == nil || msg.Chat == nil {
		return nil
	}

	if msg.Chat.Username != os.Getenv("TELEGRAM_CATIA_CHANNEL_USERNAME") {
		return nil
	}

	dbRedis, err := getContextRedis(c)
	if err != nil {
		return err
	}

	message := &models.LastMessage{
		ID:              msg.ID,
		TimeLastMessage: time.Now(),
		FreebieName:     models.NAME_GEM,
	}

	if err := redis_store.SetLastMessage(context.Background(), dbRedis, message); err != nil {
		return err
	}

	return nil
}

func commandCheckRef(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}
	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	query := c.Args()
	if len(query) < 1 {
		return c.Send("Please enter the user id or username you want to check!")
	}

	userId := query[0]
	userIdInt, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		user, err := datastore.FindUserByUsername(context.Background(), postgresDb, userId)
		if err != nil {
			return c.Send("Error when find user by username: " + err.Error())
		}

		if user == nil {
			return c.Send("User not found")
		}

		userIdInt = user.ID
	}

	count, err := datastore.CountInviteesByUserId(context.Background(), postgresDb, userIdInt)
	if err != nil {
		return c.Send("Error when get count invitees: " + err.Error())
	}

	return c.Send(fmt.Sprintf("User %s has %d invitees", userId, count))
}

func commandCheckTop10Ref(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}
	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	kols, err := datastore.CountTopInvitedUsers(context.Background(), postgresDb, 10)
	if err != nil {
		return c.Send("Error when get top invited KOLs: " + err.Error())
	}

	msg := "Top invited KOLs:"

	for index, kol := range kols {
		if kol.Username == "" {
			msg += fmt.Sprintf("\n%d. ID: %d - Invited: %d", index+1, kol.ID, kol.Count)
		} else {
			msg += fmt.Sprintf("\n%d. %s - Invited: %d", index+1, kol.Username, kol.Count)
		}
	}

	return c.Send(msg)
}

func commandGetAllTaskSlug(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	tasks, err := datastore.GetAllSocialTasks(context.Background(), postgresDb)
	if err != nil {
		return c.Send("Error when get all tasks: " + err.Error())
	}

	msg := "All tasks:"

	for index, task := range tasks {
		msg += fmt.Sprintf("\n%d. %s", index+1, task.GameSlug)
	}

	return c.Send(msg)
}

func commandGetTaskCount(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	if len(c.Args()) < 1 {
		return c.Send("Please enter the task slug you want to check!")
	}

	postgresDb, err := getContextPostgres(c)
	if err != nil {
		return c.Send(fmt.Sprintf("error %s", err.Error()))
	}

	slug := c.Args()[0]

	if slug == "all" && len(c.Args()) < 2 {
		tasks, err := datastore.GetAllSocialTasks(context.Background(), postgresDb)
		if err != nil {
			return c.Send("Error when get all tasks: " + err.Error())
		}

		var linkCounts []LinkCount

		msg := ""

		for _, t := range tasks {
			for _, l := range t.Links {
				actionTask := fmt.Sprintf("social_task:%s:%s", t.GameSlug, l.Url)
				countByLink, err := datastore.CountByAction(context.Background(), postgresDb, actionTask)
				if err != nil {
					return c.Send("Error when count task: " + err.Error())
				}

				linkCounts = append(linkCounts, LinkCount{Description: l.Description, Count: countByLink})
			}
		}

		sort.Slice(linkCounts, func(i, j int) bool {
			return linkCounts[i].Count > linkCounts[j].Count
		})

		for _, lc := range linkCounts {
			msg += fmt.Sprintf("- %s: %d.\n", lc.Description, lc.Count)
		}

		return c.Send(msg)
	}

	if len(c.Args()) >= 2 {
		periodStr := c.Args()[1]
		if len(periodStr) < 2 {
			return c.Send("Invalid period")
		}
		//take the last character
		period := periodStr[len(periodStr)-1:]
		//take the number
		periodNumber := periodStr[:len(periodStr)-1]
		periodInt, _ := strconv.Atoi(periodNumber)
		if period == "d" {
			from := time.Now().AddDate(0, 0, -periodInt)

			var linkCounts []LinkCount

			if slug == "all" {
				tasks, err := datastore.GetAllSocialTasks(context.Background(), postgresDb)
				if err != nil {
					return c.Send("Error when get all tasks: " + err.Error())
				}

				for _, t := range tasks {
					for _, l := range t.Links {
						actionTask := fmt.Sprintf("social_task:%s:%s", t.GameSlug, l.Url)
						countByLink, err := datastore.CountByActionFromTime(context.Background(), postgresDb, actionTask, &from)
						if err != nil {
							return c.Send("Error when count task: " + err.Error())
						}

						linkCounts = append(linkCounts, LinkCount{Description: l.Description, Count: countByLink})
					}
				}
			} else {
				task, err := datastore.GetSocialTask(context.Background(), postgresDb, slug)
				if err != nil {
					return c.Send("Error when get all tasks: " + err.Error())
				}

				for _, t := range task.Links {
					actionTask := fmt.Sprintf("social_task:%s:%s", slug, t.Url)
					countByLink, err := datastore.CountByActionFromTime(context.Background(), postgresDb, actionTask, &from)
					if err != nil {
						return c.Send("Error when count task: " + err.Error())
					}

					linkCounts = append(linkCounts, LinkCount{Description: t.Description, Count: countByLink})
				}
			}

			// Sort the slice by count in ascending order
			sort.Slice(linkCounts, func(i, j int) bool {
				return linkCounts[i].Count > linkCounts[j].Count
			})

			// Build the message
			msg := ""
			for _, lc := range linkCounts {
				msg += fmt.Sprintf("- %s: %d.\n", lc.Description, lc.Count)
			}

			return c.Send(msg)
		}
	}

	task, err := datastore.GetSocialTask(context.Background(), postgresDb, slug)
	if err != nil {
		return c.Send("Error when get all tasks: " + err.Error())
	}

	msg := ""

	for _, t := range task.Links {
		actionTask := fmt.Sprintf("social_task:%s:%s", slug, t.Url)
		countByLink, err := datastore.CountByAction(context.Background(), postgresDb, actionTask)
		if err != nil {
			return c.Send("Error when count task: " + err.Error())
		}

		msg += fmt.Sprintf("- %s: %d.\n", t.Description, countByLink)
	}

	return c.Send(msg)
}

func commandCheckAdminOfChannel(c tele.Context) error {
	if !AuthRequire(c, chatId) {
		return nil
	}

	if len(c.Args()) < 1 {
		return c.Send("Please enter the group chat id")
	}

	groupChatIdString := c.Args()[0]
	groupChatIdInt64, err := strconv.ParseInt(groupChatIdString, 10, 64)
	if err != nil {
		return c.Send("Invalid group chat id")
	}

	chat := tele.Chat{ID: groupChatIdInt64}
	//check if bot is admin of the group
	admins, err := c.Bot().AdminsOf(&chat)
	if err != nil {
		return c.Send("Bot is not admin of the group: " + err.Error())
	}

	for _, admin := range admins {
		if admin.User.ID == c.Bot().Me.ID {
			return c.Send("Bot is admin of the group")
		}
	}

	return c.Send("Bot is not admin of the group")
}
