package redis_store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"millionaire/internal"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	SOCIAL_CHECK_LIMIT    = 5
	SOCIAL_CHECK_COLDDOWN = 2 * time.Minute
)

func dbKeyGameSession(gameSlug string, userID int64, sessionID *string, isAll bool) string {
	if isAll {
		return fmt.Sprintf("game:%s:session:%d:*", strings.ToLower(gameSlug), userID)
	}

	if sessionID != nil {
		return fmt.Sprintf("game:%s:session:%d:%s", strings.ToLower(gameSlug), userID, *sessionID)
	}

	return ""
}

func dbKeyUserSocialTaskGame(gameSlug string, userID int64) string {
	return fmt.Sprintf("user:social_task_bonus:%d:%s", userID, gameSlug)
}

func dbKeyCheckSocialTaskCount(userId int64) string {
	return fmt.Sprintf("user:check_social_task_count:%d", userId)
}

func dbKeyUserGameSession(gameSlug string, userID int64) string {
	return fmt.Sprintf("game_session:%s:%d", strings.ToLower(gameSlug), userID)
}

func dbKeyMostPlayedSession(gameId string) string {
	return fmt.Sprintf("game:%s:most_played_session", gameId)
}

func dbKeyLongestStreak(gameId string) string {
	return fmt.Sprintf("game:%s:longest_streak", gameId)
}

func dbKeyMostBonusPoint(gameId string) string {
	return fmt.Sprintf("game:%s:most_bonus_point", gameId)
}

func dbKeyMostMinusPoint(gameId string) string {
	return fmt.Sprintf("game:%s:most_minus_point", gameId)
}

func dbKeyLeaderboard(gameSlug string) string {
	return fmt.Sprintf("leaderboard:%s", strings.ToLower(gameSlug))
}

func dbKeyJoinedSocialLink(userID int64, link string) string {
	return fmt.Sprintf("user:joined_social_link:%d:%s", userID, link)
}

func dbKeyUserLastNotify(userId int64) string {
	return fmt.Sprintf("user:%d:last_notify", userId)
}

func dbKeyQuestionGroup(gameSlug string, group string) string {
	return fmt.Sprintf("game:%s:question:group:%s", strings.ToLower(gameSlug), group)
}

func dbKeyUserMoonGacha(userID int64, moonTime time.Time) string {
	return fmt.Sprintf("user:%d:moon-gacha:%s", userID, moonTime.Format("2006-01-02 15:04:05"))
}

func dbKeyMoon() string {
	return "event:moon-gacha"
}

func dbKeyInvoiceMessage(invoceId string) string {
	return fmt.Sprintf("invoice-test:%s", strings.ToLower(invoceId))
}

func dbKeySendMessageUser(userId int64) string {
	return fmt.Sprintf("sent_message:%d", userId)
}

func GetQuestionGroup(ctx context.Context, cmd redis.Cmdable, gameSlug string, group string) ([]string, error) {
	return cmd.SMembers(ctx, dbKeyQuestionGroup(gameSlug, group)).Result()
}

func RandomQuestionFromGroup(ctx context.Context, cmd redis.Cmdable, gameSlug string, group string) string {
	return cmd.SRandMember(ctx, dbKeyQuestionGroup(gameSlug, group)).Val()
}

func AddQuestionsToGroup(ctx context.Context, cmd redis.Cmdable, gameSlug string, group string, questionIDs []int) error {
	err := cmd.Del(ctx, dbKeyQuestionGroup(gameSlug, group)).Err()
	if err != nil {
		return err
	}

	qs := make([]any, len(questionIDs))
	for i, v := range questionIDs {
		qs[i] = v
	}

	err = cmd.SAdd(ctx, dbKeyQuestionGroup(gameSlug, group), qs...).Err()
	if err != nil {
		return err
	}

	return cmd.Expire(ctx, dbKeyQuestionGroup(gameSlug, group), time.Hour).Err()
}

func DeleteGameQuestionGroups(ctx context.Context, cmd redis.UniversalClient, gameSlug string) error {
	return caching.DeleteKeys(ctx, cmd, dbKeyQuestionGroup(gameSlug, "*"))
}

// deprecated
func GetGameSessionByID(ctx context.Context, cmd redis.Cmdable, gameSlug string, userID int64, sessionID string) (*internal.GameSession, error) {
	var v *internal.GameSession
	b, err := cmd.Get(ctx, dbKeyGameSession(gameSlug, userID, &sessionID, false)).Bytes()
	if err != nil {
		return nil, err
	}

	err = msgpack.Unmarshal(b, &v)
	if err != nil {
		// try legacy mode
		var vLegacy *internal.GameSessionLegacy
		err = msgpack.Unmarshal(b, &vLegacy)
		if err == nil {
			v = vLegacy.ToGameSession()
			return v, nil
		}
	}
	return v, err
}

func GetCurrentGameSessionByUser(ctx context.Context, cmd redis.Cmdable, gameSlug string, userID int64) (*models.GameSession, error) {
	var v *models.GameSession
	b, err := cmd.Get(ctx, dbKeyUserGameSession(gameSlug, userID)).Bytes()
	if err != nil {
		return nil, err
	}

	err = msgpack.Unmarshal(b, &v)
	return v, err
}

func SaveGameSession(ctx context.Context, cmd redis.Cmdable, v *models.GameSession) (*models.GameSession, error) {
	if v.GameSlug == "" || v.UserID == 0 {
		return nil, errors.New("invalid session")
	}

	b, err := msgpack.Marshal(v)
	if err != nil {
		return nil, err
	}

	err = cmd.Set(ctx, dbKeyUserGameSession(v.GameSlug, v.UserID), b, 0).Err()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// func PersistGameSession(ctx context.Context, cmd redis.Cmdable, v *internal.GameSession) (*internal.GameSession, error) {
// 	if v.Key == "" {
// 		return nil, errors.New("invalid session id")
// 	}

// 	b, err := msgpack.Marshal(v)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = cmd.Set(ctx, v.Key, b, 0).Err()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return v, nil
// }

// deprecated
func GetGameListSessions(ctx context.Context, cmd redis.Cmdable, gameSlug string, userID int64) ([]*internal.GameSession, error) {
	var gameSessions []*internal.GameSession

	iter := cmd.Scan(ctx, 0, dbKeyGameSession(gameSlug, userID, nil, true), 0).Iterator()

	for iter.Next(ctx) {
		var v *internal.GameSession
		b, err := cmd.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			return nil, err
		}
		err = msgpack.Unmarshal(b, &v)
		if err != nil {
			// try legacy mode
			var vLegacy *internal.GameSessionLegacy
			err = msgpack.Unmarshal(b, &vLegacy)
			if err == nil {
				v = vLegacy.ToGameSession()
			}
		}

		if v != nil {
			gameSessions = append(gameSessions, v)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return gameSessions, nil
}

func SetUserLastNotify(ctx context.Context, cmd redis.Cmdable, userID int64, lastNotify time.Time) error {
	err := cmd.Set(ctx, dbKeyUserLastNotify(userID), lastNotify, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetUserLastNotify(ctx context.Context, cmd redis.Cmdable, userID int64) (time.Time, error) {
	result, err := cmd.Get(ctx, dbKeyUserLastNotify(userID)).Result()
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, result)
}

func SetLeaderboard(ctx context.Context, cmd redis.Cmdable, gameSlug string, v *models.LeaderboardItem) (*models.LeaderboardItem, error) {
	err := cmd.ZAdd(ctx, dbKeyLeaderboard(gameSlug), redis.Z{
		Score:  v.Score,
		Member: v.UserId,
	}).Err()

	if err != nil {
		return nil, err
	}

	return v, nil
}

func ClearLeaderboard(ctx context.Context, cmd redis.Cmdable, gameSlug string) error {
	err := cmd.Del(ctx, dbKeyLeaderboard(gameSlug)).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetLeaderboard(ctx context.Context, cmd redis.Cmdable, gameSlug string, num int) ([]*models.LeaderboardItem, error) {
	// num always greater than 0
	items, err := cmd.ZRevRangeWithScores(ctx, dbKeyLeaderboard(gameSlug), 0, int64(num-1)).Result()
	if err != nil {
		return nil, err
	}

	var results []*models.LeaderboardItem
	for i, item := range items {
		id, _ := strconv.ParseInt(item.Member.(string), 10, 64)
		results = append(results, &models.LeaderboardItem{
			UserId: id,
			Score:  item.Score,
			Rank:   i + 1,
		})
	}

	return results, nil
}

func GetLeaderboardWithScoreThreshold(ctx context.Context, cmd redis.Cmdable, gameSlug string, scoreThreshold int) (int, error) {
	items, err := cmd.ZRangeByScore(ctx, dbKeyLeaderboard(gameSlug), &redis.ZRangeBy{
		Min: fmt.Sprintf("(%d", scoreThreshold),
		Max: "+inf",
	}).Result()

	if err != nil {
		return 0, err
	}

	return len(items), nil
}

func GetLeaderboardPaticipantsCount(ctx context.Context, cmd redis.Cmdable, gameSlug string) (int64, error) {
	count, err := cmd.ZCard(ctx, dbKeyLeaderboard(gameSlug)).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetCurrentUpdateUser(ctx context.Context, cmd redis.Cmdable, scoreThreshold int, comparedSeconds int, gameSlug string) []*models.User {
	var users []*models.User
	iter := cmd.Scan(ctx, 0, "user:*", 0).Iterator()
	for iter.Next(ctx) {
		var v *models.User
		b, err := cmd.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			return nil
		}
		err = msgpack.Unmarshal(b, &v)
		if err != nil {
			return nil
		}

		rank, err := GetRankWithScore(ctx, cmd, gameSlug, v)
		if err != nil {
			return nil
		}

		if int(rank.Score) >= scoreThreshold && time.Since(v.UpdatedAt) > time.Duration(comparedSeconds)*time.Second {
			users = append(users, v)
		}
	}
	if err := iter.Err(); err != nil {
		return nil
	}

	return users
}

func GetRankWithScore(ctx context.Context, cmd redis.Cmdable, gameSlug string, user *models.User) (redis.RankScore, error) {
	rank, err := cmd.ZRevRankWithScore(ctx, dbKeyLeaderboard(gameSlug), strconv.FormatInt(user.ID, 10)).Result()
	if err != nil {
		return redis.RankScore{}, err
	}

	return rank, nil
}

func GetScore(ctx context.Context, cmd redis.Cmdable, gameSlug string, user *models.User) (float64, error) {
	score, err := cmd.ZScore(ctx, dbKeyLeaderboard(gameSlug), strconv.FormatInt(user.ID, 10)).Result()
	if err != nil {
		return -1, err
	}

	return score, nil

}

func GetRank(ctx context.Context, cmd redis.Cmdable, gameSlug string, user *models.User) (int64, error) {
	rank, err := cmd.ZRevRank(ctx, dbKeyLeaderboard(gameSlug), strconv.FormatInt(user.ID, 10)).Result()
	if err != nil {
		return -1, err
	}

	return rank, nil
}

func GetJoinSocial(ctx context.Context, cmd redis.Cmdable, userID int64, socialLink string) (bool, error) {
	_, err := cmd.Get(ctx, dbKeyJoinedSocialLink(userID, socialLink)).Bytes()
	if err != nil {
		return false, err
	}

	return true, err
}

func SetJoinSocial(ctx context.Context, cmd redis.Cmdable, userID int64, socialLink string) error {
	err := cmd.Set(ctx, dbKeyJoinedSocialLink(userID, socialLink), true, 0).Err()
	if err != nil {
		return err
	}

	return err
}

func SetMostPlayedSession(ctx context.Context, cmd redis.Cmdable, gameSlug string, v *models.MostSessions) error {
	err := cmd.ZAdd(ctx, dbKeyMostPlayedSession(gameSlug), redis.Z{
		Score:  float64(v.TotalSessions),
		Member: v.Username,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetMostPlayedSession(ctx context.Context, cmd redis.Cmdable, gameSlug string, num int) ([]*models.MostSessions, error) {
	items, err := cmd.ZRevRangeWithScores(ctx, dbKeyMostPlayedSession(gameSlug), 0, int64(num-1)).Result()
	if err != nil {
		return nil, err
	}

	var results []*models.MostSessions
	for _, item := range items {
		results = append(results, &models.MostSessions{
			Username:      item.Member.(string),
			TotalSessions: int(item.Score),
		})
	}

	return results, nil
}

func SetLongestStreak(ctx context.Context, cmd redis.Cmdable, gameSlug string, v *models.LongestStreak) error {
	err := cmd.ZAdd(ctx, dbKeyLongestStreak(gameSlug), redis.Z{
		Score:  float64(v.StreakPoint),
		Member: v.Username,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetLongestStreak(ctx context.Context, cmd redis.Cmdable, gameSlug string, num int) ([]*models.LongestStreak, error) {
	items, err := cmd.ZRevRangeWithScores(ctx, dbKeyLongestStreak(gameSlug), 0, int64(num-1)).Result()
	if err != nil {
		return nil, err
	}

	var results []*models.LongestStreak
	for _, item := range items {
		results = append(results, &models.LongestStreak{
			Username:    item.Member.(string),
			StreakPoint: int(item.Score),
		})
	}

	return results, nil
}

func SetMostBonusPoint(ctx context.Context, cmd redis.Cmdable, gameSlug string, v *models.MostBonusPoint) error {
	err := cmd.ZAdd(ctx, dbKeyMostBonusPoint(gameSlug), redis.Z{
		Score:  float64(v.BonusPointsTime),
		Member: v.Username,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetMostBonusPoint(ctx context.Context, cmd redis.Cmdable, gameSlug string, num int) ([]*models.MostBonusPoint, error) {
	items, err := cmd.ZRevRangeWithScores(ctx, dbKeyMostBonusPoint(gameSlug), 0, int64(num-1)).Result()
	if err != nil {
		return nil, err
	}

	var results []*models.MostBonusPoint
	for _, item := range items {
		results = append(results, &models.MostBonusPoint{
			Username:        item.Member.(string),
			BonusPointsTime: int(item.Score),
		})
	}

	return results, nil
}

func SetMostMinusPoint(ctx context.Context, cmd redis.Cmdable, gameSlug string, v *models.MostMinusPoint) error {
	err := cmd.ZAdd(ctx, dbKeyMostMinusPoint(gameSlug), redis.Z{
		Score:  float64(v.MinusPointsTime),
		Member: v.Username,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetMostMinusPoint(ctx context.Context, cmd redis.Cmdable, gameSlug string, num int) ([]*models.MostMinusPoint, error) {
	items, err := cmd.ZRevRangeWithScores(ctx, dbKeyMostMinusPoint(gameSlug), 0, int64(num-1)).Result()
	if err != nil {
		return nil, err
	}

	var results []*models.MostMinusPoint
	for _, item := range items {
		results = append(results, &models.MostMinusPoint{
			Username:        item.Member.(string),
			MinusPointsTime: int(item.Score),
		})
	}

	return results, nil
}

func CheckSocialTaskLimit(ctx context.Context, cmd redis.Cmdable, userID int64) (bool, error) {
	number, err := cmd.Get(ctx, dbKeyCheckSocialTaskCount(userID)).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	numberInt := 0
	if err != redis.Nil {
		numberInt, _ = strconv.Atoi(number)
	}

	if numberInt <= SOCIAL_CHECK_LIMIT {
		_, err = cmd.Incr(ctx, dbKeyCheckSocialTaskCount(userID)).Result()
		if err != nil {
			return false, err
		}

		err = cmd.Expire(ctx, dbKeyCheckSocialTaskCount(userID), SOCIAL_CHECK_COLDDOWN).Err()
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func SetUserSocialTaskGame(ctx context.Context, cmd redis.Cmdable, gameSlug string, userID int64) (bool, error) {
	err := cmd.Set(ctx, dbKeyUserSocialTaskGame(gameSlug, userID), true, 0).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetUserSocialTaskGame(ctx context.Context, cmd redis.Cmdable, gameSlug string, userID int64) (bool, error) {
	_, err := cmd.Get(ctx, dbKeyUserSocialTaskGame(gameSlug, userID)).Bytes()
	if err != nil {
		return false, err
	}

	return true, nil
}

func SetLastMessage(ctx context.Context, cmd redis.Cmdable, lastMessage *models.LastMessage) error {
	b, err := msgpack.Marshal(lastMessage)
	if err != nil {
		return err
	}

	err = cmd.Set(ctx, dbKeyLastMessage(), b, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func dbKeyLastMessage() string {
	return "last_message"
}

func GetLastMessage(ctx context.Context, cmd redis.Cmdable) (*models.LastMessage, error) {
	var v *models.LastMessage
	b, err := cmd.Get(ctx, dbKeyLastMessage()).Bytes()
	if err != nil {
		return nil, err
	}

	err = msgpack.Unmarshal(b, &v)
	return v, err
}

func GetSIWTNonce(ctx context.Context, cmd redis.Cmdable, key string) (string, error) {
	n, err := cmd.Get(ctx, key).Result()
	if err != nil {
		return n, err
	}

	return n, err
}

func SetSIWTNonce(ctx context.Context, cmd redis.Cmdable, key, nonce string, expiration time.Duration) error {
	err := cmd.Set(ctx, key, nonce, expiration).Err()
	if err != nil {
		return err
	}

	return err
}

func SetMoon(ctx context.Context, cmd redis.Cmdable, v *models.Moon) error {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}

	err = cmd.Set(ctx, dbKeyMoon(), b, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetMoon(ctx context.Context, cmd redis.Cmdable) (*models.Moon, error) {
	var v *models.Moon
	b, err := cmd.Get(ctx, dbKeyMoon()).Bytes()
	if err != nil {
		return nil, err
	}

	err = msgpack.Unmarshal(b, &v)
	return v, err
}

func SetUserMoonGacha(ctx context.Context, cmd redis.Cmdable, userID int64, moonTime time.Time) error {
	err := cmd.Set(ctx, dbKeyUserMoonGacha(userID, moonTime), true, time.Hour*24).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetUserMoonGacha(ctx context.Context, cmd redis.Cmdable, userID int64, moonTime time.Time) (bool, error) {
	_, err := cmd.Get(ctx, dbKeyUserMoonGacha(userID, moonTime)).Bytes()
	if err != nil {
		return false, err
	}

	return true, nil
}

func SetInvoiceMessage(ctx context.Context, cmd redis.Cmdable, invoiceId string, message *tele.StoredMessage) error {
	b, err := msgpack.Marshal(message)
	if err != nil {
		return err
	}

	err = cmd.Set(ctx, dbKeyInvoiceMessage(invoiceId), b, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetInvoiceMessage(ctx context.Context, cmd redis.Cmdable, invoiceId string) (*tele.StoredMessage, error) {
	var v *tele.StoredMessage
	b, err := cmd.Get(ctx, dbKeyInvoiceMessage(invoiceId)).Bytes()
	if err != nil {
		return nil, err
	}

	err = msgpack.Unmarshal(b, &v)
	return v, err
}

func SetSendMessageUser(ctx context.Context, cmd redis.Cmdable, userId int64) error {
	err := cmd.Set(ctx, dbKeySendMessageUser(userId), true, time.Hour*24).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetSendMessageUser(ctx context.Context, cmd redis.Cmdable, userId int64) (bool, error) {
	_, err := cmd.Get(ctx, dbKeySendMessageUser(userId)).Bytes()
	if err != nil {
		return false, err
	}

	return true, nil
}
