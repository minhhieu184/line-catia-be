package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrGameSessionLock = errors.New("session game locked")
var ErrUserBoostLock = errors.New("user boost locked")
var ErrUserGameLock = errors.New("user game locked")
var ErrFullMoonLock = errors.New("full moon locked")

const (
	CONFIG_SERVER_MODE                    = "SERVER_MODE"
	CONFIG_GAME_LEADERBOARD_LIMIT         = "GAME_LEADERBOARD_LIMIT"
	CONFIG_REFERRAL_LEADERBOARD_LIMIT     = "REF_LEADERBOARD_LIMIT"
	CONFIG_OVERALL_LEADERBOARD_LIMIT      = "OVERALL_LEADERBOARD_LIMIT"
	CONFIG_ARENA_LEADERBOARD_LIMIT        = "ARENA_LEADERBOARD_LIMIT"
	CONFIG_BONUS_TASK_POINT               = "BONUS_TASK_POINT"
	CONFIG_TEXT_NEW_USER                  = "TEXT_NEW_USER"
	CONFIG_MIN_GEM_TO_CLAIM_REF_BOOST     = "MIN_GEM_TO_CLAIM_REF_BOOST"
	CONFIG_FREEBIE_GEM_COUNTDOWN          = "FREEBIE_GEM_COUNTDOWN"
	CONFIG_FREEBIE_STAR_COUNTDOWN         = "FREEBIE_STAR_COUNTDOWN"
	CONFIG_FREEBIE_LIFELINE_COUNTDOWN     = "FREEBIE_LIFELINE_COUNTDOWN"
	CONFIG_CRONJOB_TIME_FULL_MOON         = "CRONJOB_TIME_FULL_MOON"
	CONFIG_FULL_MOON_START_TIME           = "FULL_MOON_START_TIME"
	CONFIG_FULL_MOON_END_TIME             = "FULL_MOON_END_TIME"
	CONFIG_MOON_TIME_PER_RANGE_IN_MINUTES = "MOON_TIME_PER_RANGE_IN_MINUTES"
	CONFIG_MOON_EXPIRED_TIME_IN_MINUTES   = "MOON_EXPIRED_TIME_IN_MINUTES"
	CONFIG_MOON_RANDOM_UNIT_IN_MINUTES    = "MOON_RANDOM_UNIT_IN_MINUTES"

	SERVER_MODE_DEVELOPMENT = "development"
	SERVER_MODE_STAGING     = "staging"
	SERVER_MODE_PRODUCTION  = "production"

	LEADERBOARD_OVERALL        = "overall"
	LEADERBOARD_OVERALL_WEEKLY = "overall_weekly"
	LEADERBOARD_REFERRAL       = "referral"

	GAME_LEADERBOARD_DEFAULT_LIMIT           = 20
	REFERRAL_LEADERBOARD_DEFAULT_LIMIT       = 20
	OVERALL_LEADERBOARD_DEFAULT_LIMIT        = 20
	DEFAULT_SESSION_COUNTDOWN_IN_MINUTES     = 10
	DEFAULT_TIME_REDUCE_PER_BOOST_IN_MINUTES = 5

	CACHE_TTL_5_SECONDS  = 5 * time.Second
	CACHE_TTL_15_SECONDS = 15 * time.Second
	CACHE_TTL_1_MIN      = 1 * time.Minute
	CACHE_TTL_5_MINS     = 5 * time.Minute
	CACHE_TTL_15_MINS    = 15 * time.Minute
	CACHE_TTL_30_MINS    = 30 * time.Minute
	CACHE_TTL_1_HOUR     = 1 * time.Hour
	CACHE_TTL_1_DAY      = 24 * time.Hour

	TELEGRAM_API_BASE_URL = "https://api.telegram.org"

	TELEGRAM_TASK_RATE_LIMIT_PER_MINUTE = 10
	PARTNER_RATE_LIMIT_PER_MINUTE       = 10000

	LIFELINES_PER_STAR = 3

	MIN_GEM_TO_CLAIM_REF_BOOST = 16

	KEY_SOCIAL_TASK = "social_task:%s:%s"

	TELETOP_CATIA_APP_ID = 143

	LINE_API_BASE_URL = "https://api.line.me/oauth2/v2.1"
)

func LockKeyUserGameSession(gameSlug string, userID string) string {
	return fmt.Sprintf("user_game_session:lock:%s:%d", gameSlug, userID)
}

func LockKeyUserBoost(userID string) string {
	return fmt.Sprintf("lock:user-boost:%d", userID)
}

func LockKeyUserClaimBoost(userID string, source string) string {
	return fmt.Sprintf("lock:user-claim-boost:%d,%s", userID, source)
}

func LockKeyUserClaimAllBoost(userID string) string {
	return fmt.Sprintf("lock:user-claim-all-boost:%d", userID)
}

func LockKeyUserGame(gameSlug string, userID string) string {
	return fmt.Sprintf("lock:user-game:%s:%d", gameSlug, userID)
}

func LockKeyUserMoon(userID string) string {
	return fmt.Sprintf("lock:user-moon:%d", userID)
}

func LockKeyFullMoon() string {
	return "lock:full-moon"
}

// db
func DBKeyGame(gameSlug string) string {
	return fmt.Sprintf("game:%s", strings.ToLower(gameSlug))
}

func DBKeyGames() string {
	return "games:active"
}

// db
func DBKeyUser(userID string) string {
	return fmt.Sprintf("user:%d", userID)
}

func DBKeyMe(userID string) string {
	return fmt.Sprintf("me:%d", userID)
}

func DBKeyConfig(key string) string {
	return fmt.Sprintf("config:%s", strings.ToLower(key))
}

func DBKeyLeaderboardByUser(name string, userID string, limit int) string {
	return fmt.Sprintf("leaderboard_by_user:%s:%d:%d", strings.ToLower(name), userID, limit)
}

func DBKeyUserGameSessionSumary(gameSlug string, userID string) string {
	return fmt.Sprintf("user_game:sumary:%s:%d", gameSlug, userID)
}

func DBKeyLastUserGameSession(gameSlug string, userID string) string {
	return fmt.Sprintf("user_game:last_session:%s:%d", gameSlug, userID)
}

func DBKeyGameCategory(gameSlug string) string {
	return fmt.Sprintf("game-category:%s", strings.ToLower(gameSlug))
}

func DBKeyAllSocialTasks() string {
	return "social_task:all"
}

func DBKeyGameSocialTasks(gameSlug string) string {
	return fmt.Sprintf("social_task:game:%s", gameSlug)
}

func DBKeyUserPassedSocialTask(userID string, gameSlug string) string {
	return fmt.Sprintf("social_task:user:%d:%s:passed", userID, gameSlug)
}

func DBKeyUserSocialBonusScore(userID string) string {
	return fmt.Sprintf("user:%d:social_bonus_score", userID)
}

func DBKeyUserGemAction(userID string, action string) string {
	return fmt.Sprintf("user_gem:%d:%s", userID, action)
}

func DBKeySocialLink(linkID int, gameSlug string) string {
	return fmt.Sprintf("social_link:%d:%s", linkID, gameSlug)
}

func DBKeyUserSocialTasks(userID string, gameSlug string) string {
	return fmt.Sprintf("social_task:user:%d:%s", userID, gameSlug)
}

func DBKeyUserSocialTask(userID string) string {
	return fmt.Sprintf("social_task:user:%d", userID)
}

func DBKeyGameConfig(gameSlug, key string) string {
	return fmt.Sprintf("game-config:%s:%s", gameSlug, strings.ToLower(key))
}

func DBKeyGameCountdownTime(gameSlug string) string {
	return fmt.Sprintf("game:countdown_time:%s", gameSlug)
}

func DBKeyGameReduceTimePerBoost(gameSlug string) string {
	return fmt.Sprintf("game:reduce_time_per_boost:%s", gameSlug)
}

func DBKeyUserGame(gameSlug string, userID string) string {
	return fmt.Sprintf("user_game:%s:%d", gameSlug, userID)
}

func DBKeyQuestion(questionID int) string {
	return fmt.Sprintf("question:%d", questionID)
}

func DBKeyUserFreebies(userID string, action string) string {
	return fmt.Sprintf("user_freebies:%d:%s", userID, action)
}

func DBKeyUserAllFreebies(userID string) string {
	return fmt.Sprintf("user_freebies:%d", userID)
}

func DBKeyUserGems(userID string) string {
	return fmt.Sprintf("user_gems:%d", userID)
}

func DBKeyUserSocialTaskVerify(userID string, gameslug string, url string) string {
	return fmt.Sprintf("user:verify_join_social_link:%d:%s:%s", userID, gameslug, url)
}

func DBKeyUserFriendList(userID string, page int, limit int) string {
	return fmt.Sprintf("user_friend_list:%d:%d:%d", userID, page, limit)
}

func DBKeyUserWallet(userID string) string {
	return fmt.Sprintf("user_wallet:%d", userID)
}

func DBKeyBoostExist(userID string, source string) string {
	return fmt.Sprintf("boost:exist:%d:%s", userID, source)
}

func DBKeyPartner(slug string) string {
	return fmt.Sprintf("partner:%s", slug)
}

func DBKeyUserJoined(userID string, refCode string, minGem int) string {
	return fmt.Sprintf("user_joined:%s:%d:%d", userID, refCode, minGem)
}

func DBKeyUserByRefCode(refCode string) string {
	return fmt.Sprintf("user:by_ref_code:%s", refCode)
}

func DBKeyUserAvailableReward(userID string) string {
	return fmt.Sprintf("user:available_rewards:%d", userID)
}

func LimitKeyUserSocialTask(userID string) string {
	return fmt.Sprintf("users:social_task:%d", userID)
}

func LimitKeyParner(slug string) string {
	return fmt.Sprintf("limit:partner:%s", slug)
}

func DBKeyArenaList() string {
	return "arena:list"
}

func DBKeyArena(slug string) string {
	return fmt.Sprintf("arena:%s", slug)
}

func DBKeyArenaByGameSlug(gameSlug string) string {
	return fmt.Sprintf("arena:game_slug:%s", gameSlug)
}

func DBKeyFriendCount(userID string) string {
	return fmt.Sprintf("friend_count:%d", userID)
}
