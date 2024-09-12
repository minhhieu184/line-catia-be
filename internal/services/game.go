package services

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mroth/weightedrand/v2"

	"millionaire/internal"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg"
	"millionaire/internal/pkg/caching"

	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceGame struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache

	serviceUser        *ServiceUser
	serviceSocial      *ServiceSocial
	serviceUserGame    *ServiceUserGame
	serviceQuestion    *ServiceQuestion
	serviceConfig      *ServiceConfig
	serviceLeaderboard *ServiceLeaderboard
	//gacha           *ServiceGacha[models.ExtraSetupType]

}

const (
	STREAK_STEP_POINT       = "streak_point"
	TIME_COUNTDOWN          = "countdown_time"
	NUMBER_OF_BOOST_ALLOWED = "number_of_boost_allowed"
	TIME_REDUCE_PER_BOOST   = "time_reduce_per_boost"
)

func NewServiceGame(container *do.Injector) (*ServiceGame, error) {
	redisDB, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	postgresDB, err := do.Invoke[*bun.DB](container)
	if err != nil {
		return nil, err
	}

	cache, err := do.Invoke[caching.Cache](container)
	if err != nil {
		return nil, err
	}

	serviceUser, err := do.Invoke[*ServiceUser](container)
	if err != nil {
		return nil, err
	}

	serviceSocial, err := do.Invoke[*ServiceSocial](container)
	if err != nil {
		return nil, err
	}

	serviceUserGame, err := do.Invoke[*ServiceUserGame](container)
	if err != nil {
		return nil, err
	}

	serviceQuestion, err := do.Invoke[*ServiceQuestion](container)
	if err != nil {
		return nil, err
	}

	serviceConfig, err := do.Invoke[*ServiceConfig](container)
	if err != nil {
		return nil, err
	}

	serviceLeaderboard, err := do.Invoke[*ServiceLeaderboard](container)
	if err != nil {
		return nil, err
	}

	rs, err := do.Invoke[*redsync.Redsync](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	readOnlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	return &ServiceGame{container, redisDB, rs, postgresDB, readonlyPostgresDB, cache, readOnlyCache, serviceUser, serviceSocial, serviceUserGame, serviceQuestion, serviceConfig, serviceLeaderboard}, nil
}

func (service *ServiceGame) FindOrCreateSession(ctx context.Context, gameSlug string, user *models.User) (*models.GameSession, error) {
	game, err := service.GetGame(ctx, gameSlug)
	if err != nil {
		return nil, errorx.Wrap(errors.New("game not found"), errorx.NotExist)
	}

	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if userGame == nil {
		return nil, err
	}

	println("user id", user.ID)
	mutex := service.rs.NewMutex(LockKeyUserGameSession(game.Slug, user.ID))
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrGameSessionLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	// Attempt to retrieve current session
	currentSession, err := service.GetCurrentGameSession(ctx, userGame)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// Generate a unique session Key
	bytes, _ := hex.DecodeString(uuid.New().String())
	newSessionID := fmt.Sprintf("0x%x", bytes)

	// If the user doesn't have a current session or is new
	if err == redis.Nil {
		// Persist user changes
		_, err = service.serviceUserGame.UpdateCountdownAndExtraSession(ctx, userGame, time.Now(), userGame.ExtraSessions)
		if err != nil {
			return nil, err
		}

		// Create a new session
		newSession := &models.GameSession{
			LegacyID:          newSessionID,
			GameSlug:          gameSlug,
			UserID:            user.ID,
			NextStep:          0,
			CurrentQuestion:   nil,
			Score:             0,
			QuestionStartedAt: nil,
			StreakPoint:       0,
			UsedBoostCount:    0,
		}

		// Persist the new session
		newSession, err = redis_store.SaveGameSession(ctx, service.redisDB, newSession)
		if err != nil {
			return nil, err
		}

		return newSession, nil
	}

	// If current session has ended but not yet switched to a new session by incident
	if currentSession.EndedAt != nil {
		newSession := &models.GameSession{
			LegacyID:          newSessionID,
			GameSlug:          gameSlug,
			UserID:            user.ID,
			NextStep:          0,
			CurrentQuestion:   nil,
			Score:             0,
			QuestionStartedAt: nil,
			StreakPoint:       currentSession.StreakPoint,
			UsedBoostCount:    0,
		}

		_ = datastore.SaveGameSession(ctx, service.postgresDB, currentSession)

		newSession, err = redis_store.SaveGameSession(ctx, service.redisDB, newSession)
		if err != nil {
			return nil, err
		}

		return newSession, nil
	}

	return currentSession, nil
}

func (service *ServiceGame) switchToNewSession(ctx context.Context, userGame *models.UserGame, currentSession *models.GameSession) error {
	if err := service.cache.Delete(ctx, DBKeyUserGameSessionSumary(userGame.GameSlug, userGame.UserID)); err != nil {
		return err
	}

	if err := service.cache.Delete(ctx, DBKeyLastUserGameSession(userGame.GameSlug, userGame.UserID)); err != nil {
		return err
	}

	// Generate a unique session Key
	bytes, _ := hex.DecodeString(uuid.New().String())
	newSessionID := fmt.Sprintf("0x%x", bytes)
	// User has an existing session
	if userGame.CountdownEnded() {
		// Attempt to retrieve current session
		// If the current time is after the user's countdown, create a new session and save the current session to the database
		_, err := service.serviceUserGame.UpdateCountdownAndExtraSession(ctx, userGame, *userGame.Countdown, 0)
		if err != nil {
			return err
		}
	} else {
		// If the user has extra sessions, use one and reset streak
		if userGame.ExtraSessions > 0 {
			extras := userGame.ExtraSessions
			extras--

			_, err := service.serviceUserGame.UpdateCountdownAndExtraSession(ctx, userGame, *userGame.Countdown, extras)
			if err != nil {
				return err
			}
		}
	}

	newSession := &models.GameSession{
		LegacyID:          newSessionID,
		GameSlug:          userGame.GameSlug,
		UserID:            userGame.UserID,
		NextStep:          0,
		CurrentQuestion:   nil,
		Score:             0,
		QuestionStartedAt: nil,
		StreakPoint:       currentSession.StreakPoint,
		UsedBoostCount:    0,
	}

	err := datastore.SaveGameSession(ctx, service.postgresDB, currentSession)
	if err != nil {
		return err
	}
	_, err = redis_store.SaveGameSession(ctx, service.redisDB, newSession)
	if err != nil {
		return err
	}

	return nil
}

func (service *ServiceGame) getNextQuestion(ctx context.Context, user *models.User, game *models.Game, step int) (*models.Question, int, error) {
	// TODO: use userID to minimize duplication
	questionSetup := game.Questions[step]
	if questionSetup.Extra {
		return &models.Question{
			ID:    -1,
			Extra: true,
		}, 0, nil
	}

	if !questionSetup.Difficulty.Valid() {
		return nil, 0, errorx.Wrap(errors.New("invalid setup: difficulty"), errorx.Validation)
	}

	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return nil, 0, err
	}

	session, err := service.GetCurrentGameSession(ctx, userGame)
	if err != nil {
		return nil, 0, err
	}

	if session == nil {
		return nil, 0, errorx.Wrap(errors.New("session not found"), errorx.NotExist)
	}

	if session.GameSlug == "" {
		session.GameSlug = game.Slug
	}

	return service.serviceQuestion.RandomNextQuestion(ctx, session, &questionSetup)
}

func (service *ServiceGame) NextQuestion(ctx context.Context, gameID string, user *models.User) (*models.GameSession, error) {
	game, err := service.GetGame(ctx, gameID)
	if err == redis.Nil {
		return nil, errorx.Wrap(errors.New("game not found"), errorx.NotExist)
	}
	if err != nil {
		return nil, err
	}

	session, err := service.FindOrCreateSession(ctx, gameID, user)
	if err != nil {
		return nil, err
	}

	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return nil, err
	}

	// Game not started and countdown not ended
	if !userGame.CountdownEnded() && session.CurrentQuestion == nil {
		return nil, errorx.Wrap(errors.New("countdown not ended"), errorx.Validation)
	}

	if session.EndedAt != nil {
		return nil, errorx.Wrap(errors.New("game ended"), errorx.Validation)
	}

	if session.QuestionStartedAt != nil {
		// already given a current question
		return session, nil
	}

	// if game is belong to a arena, check all required tasks are completed before start
	if !game.IsPublic && session.NextStep == 0 {
		//check usertask here
		tasks, err := service.serviceSocial.GetUserTasks(ctx, user.ID, game.Slug)
		if err != nil {
			return nil, errorx.Wrap(errors.New("something went wrong when getting user tasks"), errorx.Validation)
		}

		for _, link := range tasks.Links {
			//if task is not completed, return error
			if link.Required && !link.Joined {
				return nil, errorx.Wrap(errors.New("required tasks not completed"), errorx.Validation)
			}
		}
	}

	mutex := service.rs.NewMutex(LockKeyUserGameSession(gameID, user.ID))
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrGameSessionLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	now := time.Now()
	lastStep := session.NextStep
	if len(game.Questions) <= lastStep {
		session.EndedAt = &now
		if session, err = redis_store.SaveGameSession(ctx, service.redisDB, session); err != nil {
			return nil, err
		}

		if _, err = service.endGame(ctx, user, userGame, game, session); err != nil {
			return nil, err
		}

		return nil, errorx.Wrap(errors.New("no more question, game ended"), errorx.Validation)
	}

	question, questionScore, err := service.getNextQuestion(ctx, user, game, lastStep)
	if err == redis.Nil {
		return nil, errorx.Wrap(errors.New("question not found"), errorx.NotExist)
	}
	if err != nil {
		return nil, err
	}

	session.NextStep = lastStep + 1
	session.CurrentQuestion = question
	session.CurrentQuestionScore = questionScore
	session.QuestionStartedAt = &now
	if session.StartedAt == nil {
		session.StartedAt = &now
	}

	history := session.History
	if history == nil {
		history = map[int]models.QuestionHistory{}
	}
	history[lastStep] = models.QuestionHistory{
		Question:      *question,
		QuestionScore: session.CurrentQuestionScore,
		TotalScore:    session.Score,
		StartedAt:     now,
	}

	session.History = history
	session, err = redis_store.SaveGameSession(ctx, service.redisDB, session)
	return session, err
}

func (service *ServiceGame) Answer(ctx context.Context, user *models.User, userGame *models.UserGame, gameAnswer models.GameAnswer) (*models.GameSession, error) {
	mutex := service.rs.NewMutex(LockKeyUserGameSession(userGame.GameSlug, user.ID))
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrGameSessionLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	gameCountdownTime := service.getGameCountdownTime(ctx, userGame.GameSlug)

	session, err := service.GetCurrentGameSession(ctx, userGame)

	if err == redis.Nil {
		return nil, errorx.Wrap(errors.New("session not found"), errorx.NotExist)
	}
	if err != nil {
		return nil, err
	}

	// TODO check question index

	lastStep := session.NextStep - 1

	if lastStep != gameAnswer.QuestionIndex {
		return session, errorx.Wrap(errors.New("wrong question index"), errorx.NotExist)
	}

	if session.QuestionStartedAt == nil {
		return session, errorx.Wrap(errors.New("session not started"), errorx.NotExist)
	}

	if session.History == nil {
		return session, errorx.Wrap(errors.New("history missing"), errorx.NotExist)
	}

	err = service.canAnswer(ctx, user, userGame, session)
	if err != nil {
		return nil, err
	}

	game, _ := service.GetGame(ctx, userGame.GameSlug)
	if game == nil {
		return nil, errorx.Wrap(errors.New("game not found"), errorx.NotExist)
	}

	now := time.Now()

	answer := gameAnswer.Answer
	correct := session.CurrentQuestion.CorrectAnswer == answer

	if session.CurrentQuestion.Extra {
		choices := []weightedrand.Choice[int, int]{}
		for i, v := range game.ExtraSetups {
			choices = append(choices, weightedrand.NewChoice(i, v.Chance))
		}
		gacha, err := NewServiceGacha[int](choices)
		if err != nil {
			return nil, err
		}

		prizeIndex := gacha.Pick()
		prize := models.ExtraSetup{
			Type: models.ExtraSetupTypeNothing.String(),
		}
		if prizeIndex >= 0 && prizeIndex < len(game.ExtraSetups) {
			prize = game.ExtraSetups[prizeIndex]
		}
		prizeType := models.ToExtraSetupType(prize.Type)

		if !prizeType.Valid() {
			return session, errorx.Wrap(errors.New("invalid outcome"), errorx.Service)
		}

		correct = true // extra => always correct
		answer = prizeIndex
		scoreAfter, anotherSession := prizeType.ToScore(session.Score)

		if anotherSession {
			// TODO: user json set extra session
			extras := userGame.ExtraSessions
			extras++

			userGame, err = service.serviceUserGame.UpdateCountdownAndExtraSession(ctx, userGame, *userGame.Countdown, extras)
			if err != nil {
				return nil, errorx.Wrap(errors.New("invalid user"), errorx.Service)
			}
		}

		session.CurrentQuestionScore = scoreAfter
		session.EndedAt = &now
		// increase streak point when user answer correctly all questions and session started at before countdown
		currentCountDownTime := *userGame.Countdown
		if session.StartedAt.Before(currentCountDownTime.Add(gameCountdownTime)) {
			session.StreakPoint += 1
		} else {
			session.StreakPoint = 1
		}

		// update assistance
		//if userGame.Lifeline == nil {
		//	userGame.Lifeline = &models.Lifeline{
		//		ChangeQuestion: false,
		//		FiftyFifty:     true,
		//	}
		//}
		//userGame.Lifeline.FiftyFifty = true
		//
		//userGame, err = datastore.UpdateUserLifeline(ctx, *service.postgresDB, userGame)
		//if err != nil {
		//	fmt.Println("UpdateUserLifeline error:", err, "user:", user.ID, "username:", user.Username)
		//}
	}

	// no matter right or wrong, we reset to mark it as answered
	session.QuestionStartedAt = nil

	history := session.History[lastStep]
	history.Answer = &answer
	history.AnsweredAt = &now
	history.Correct = &correct

	if correct {
		// correct
		session.Score = session.CurrentQuestionScore
		history.TotalScore = session.Score
		session.History[lastStep] = history

		session, err = redis_store.SaveGameSession(ctx, service.redisDB, session)
		if err != nil {
			return nil, err
		}

		if session.EndedAt != nil {
			return service.endGame(ctx, user, userGame, game, session)
		}

		return session, err
	} else {
		history.CorrectAnswer = &session.CurrentQuestion.CorrectAnswer
	}

	// incorrect
	session.EndedAt = &now
	session.StreakPoint = 0
	session.Score = 0

	// nearest checkpoint
	if err != nil {
		return nil, err
	}

	currentStep := session.NextStep - 1
	session.Score = 0

	i := currentStep - 1

	if i >= 0 {
		session.Score = game.Questions[i].Score
		history.Checkpoint = &i
	}

	history.TotalScore = session.Score
	session.History[lastStep] = history
	session, err = redis_store.SaveGameSession(ctx, service.redisDB, session)
	if err != nil {
		log.Println("SaveGameSession error:", err, "user:", user.ID, "session:", session.LegacyID)
	}

	return service.endGame(ctx, user, userGame, game, session)
}

func (service *ServiceGame) GetGame(ctx context.Context, gameSlug string) (*models.Game, error) {
	slug := strings.ToLower(gameSlug)
	callback := func() (*models.Game, error) {
		return datastore.GetGame(ctx, service.readonlyPostgresDB, slug)
	}

	game, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGame(gameSlug), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return nil, err
	}

	return game, nil
}

func (service *ServiceGame) QuitGame(ctx context.Context, gameSlug string, user *models.User) (*models.GameSession, error) {
	game, err := service.GetGame(ctx, gameSlug)
	if err != nil {
		return nil, err
	}

	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return nil, err
	}

	session, err := service.GetCurrentGameSession(ctx, userGame)

	if err == redis.Nil {
		return nil, errorx.Wrap(errors.New("session not found"), errorx.NotExist)
	}
	if err != nil {
		return nil, err
	}

	if session.EndedAt != nil {
		return nil, errorx.Wrap(errors.New("session ended"), errorx.NotExist)
	}

	now := time.Now()
	session.EndedAt = &now
	session.QuestionStartedAt = nil

	session, err = redis_store.SaveGameSession(ctx, service.redisDB, session)
	if err != nil {
		return nil, err
	}

	return service.endGame(ctx, user, userGame, game, session)
}

func (service *ServiceGame) endGame(ctx context.Context, user *models.User, userGame *models.UserGame, game *models.Game, currentSession *models.GameSession) (*models.GameSession, error) {
	gameCountdownTime := service.getGameCountdownTime(ctx, userGame.GameSlug)
	streakStepPoint, _ := service.GetGameIntConfig(ctx, userGame.GameSlug, STREAK_STEP_POINT, 20)
	if currentSession.Score > 0 {
		sessionCorrectAnswers := 0
		for _, history := range currentSession.History {
			if history.Correct != nil && *history.Correct {
				sessionCorrectAnswers++
			}
		}
		currentSession.CorrectAnswerCount = sessionCorrectAnswers
		currentSession.BonusScore = 0
	}

	currentSession.TotalScore = currentSession.Score + currentSession.BonusScore + currentSession.StreakPoint*streakStepPoint

	if err := service.switchToNewSession(ctx, userGame, currentSession); err != nil {
		return nil, err
	}

	if _, err := service.serviceUserGame.ResetCountdown(ctx, userGame, gameCountdownTime); err != nil {
		return nil, err
	}

	sessionSumary, err := service.serviceUserGame.GetUserGameSessionSumary(ctx, userGame)
	if err != nil {
		return nil, err
	}

	// update sorted set
	_, err = redis_store.SetLeaderboard(ctx, service.redisDB, userGame.GameSlug, &models.LeaderboardItem{
		UserId: userGame.UserID,
		Score:  float64(sessionSumary.TotalScore),
	})
	if err != nil {
		return nil, err
	}

	err = service.serviceUser.InsertUserGem(ctx, user, currentSession.TotalScore, fmt.Sprintf("quiz:%s:%s", userGame.GameSlug, currentSession.LegacyID))

	if err != nil {
		return nil, err
	}

	// update arena leaderboard if game is belong to an arena
	if !game.IsPublic {
		serviceArena, err := do.Invoke[*ServiceArena](service.container)
		if err != nil || serviceArena == nil {
			return currentSession, nil
		}

		arena, _ := serviceArena.GetArenaByGameSlug(ctx, userGame.GameSlug)

		if arena != nil {
			_ = serviceArena.UpdateArenaLeaderboard(ctx, user, arena)
		}
	}

	return currentSession, nil

}

func (service *ServiceGame) GetUserGameScore(ctx context.Context, userGame *models.UserGame, user *models.User) (*internal.GameScore, error) {
	sessionSumary, err := service.serviceUserGame.GetUserGameSessionSumary(ctx, userGame)
	if err != nil {
		return nil, err
	}
	currentStreak := 0
	currentScoreSession := 0
	lastMilestone := userGame.CurrentBonusMilestone
	currentSession, err := service.GetCurrentGameSession(ctx, userGame)
	if currentSession != nil {
		currentScoreSession = currentSession.Score
		currentStreak = currentSession.StreakPoint
	}

	if err != nil && err != redis.Nil {
		return nil, err
	}

	gameScore := &internal.GameScore{
		GameID:                   userGame.GameSlug,
		UserID:                   user.ID,
		TotalSessions:            sessionSumary.SessionCount,
		TotalScore:               sessionSumary.TotalScore,
		TotalScoreCurrentSession: currentScoreSession,
		CurrentStreak:            currentStreak,
		Milestone:                userGame.CurrentBonusMilestone,
		AchievedNewMilestone:     false,
		LastMilestone:            lastMilestone,
	}

	return gameScore, nil
}

func (service *ServiceGame) GetLastUserSessionScore(ctx context.Context, userGame *models.UserGame, user *models.User) (*internal.GameScore, error) {
	sessionSumary, err := service.serviceUserGame.GetUserGameSessionSumary(ctx, userGame)
	if err != nil {
		return nil, err
	}
	currentStreak := 0
	currentScoreSession := 0
	lastMilestone := userGame.CurrentBonusMilestone
	lastSession, err := service.serviceUserGame.GetLastUserGameSession(ctx, userGame)

	if err != nil && err != redis.Nil {
		return nil, err
	}

	if lastSession != nil {
		currentScoreSession = lastSession.TotalScore
		currentStreak = lastSession.StreakPoint
	}

	achieved := false

	if lastMilestone < 3 {
		if userGame.CurrentBonusMilestone < 1 && sessionSumary.MaxCorrectAnswerCount >= 5 {
			userGame.CurrentBonusMilestone += 1
			achieved = true
		}

		if userGame.CurrentBonusMilestone < 2 && sessionSumary.MaxCorrectAnswerCount >= 9 {
			userGame.CurrentBonusMilestone += 1
			achieved = true
		}

		if userGame.CurrentBonusMilestone < 3 && sessionSumary.CorrectAnswerCount >= 30 && sessionSumary.SessionCount >= 5 {
			userGame.CurrentBonusMilestone += 1
			achieved = true
		}
	}

	gameScore := &internal.GameScore{
		GameID:                   userGame.GameSlug,
		UserID:                   user.ID,
		TotalSessions:            sessionSumary.SessionCount,
		TotalScore:               sessionSumary.TotalScore,
		TotalScoreCurrentSession: currentScoreSession,
		CurrentStreak:            currentStreak,
		Milestone:                userGame.CurrentBonusMilestone,
		AchievedNewMilestone:     achieved,
		LastMilestone:            lastMilestone,
	}

	// if achieved {
	// 	_, err = datastore.UpdateUserBonusMilestone(ctx, *service.postgresDB, userGame)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	return gameScore, nil
}

func (service *ServiceGame) BurnAssistance(ctx context.Context, gameSlug string, assistanceType models.AssistanceType, user *models.User) (*models.GameSession, error) {
	game, err := service.GetGame(ctx, gameSlug)
	if err != nil {
		return nil, err
	}

	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return nil, err
	}

	mutex := service.rs.NewMutex(LockKeyUserGameSession(game.Slug, user.ID))
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrGameSessionLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	sessionUsing, err := service.GetCurrentGameSession(ctx, userGame)
	if err != nil {
		return nil, errorx.Wrap(errors.New("no session available"), errorx.NotExist)
	}

	// session null or session closed or not allow for last question
	if sessionUsing.CurrentQuestion == nil || sessionUsing.EndedAt != nil || len(sessionUsing.History) == 10 {
		return nil, errorx.Wrap(errors.New("invalid question"), errorx.Invalid)
	}

	isValid := false

	action := ""

	if assistanceType == models.AssistanceTypeFiftyFifty && user.LifelineBalance >= 1 {
		if len(sessionUsing.CurrentQuestion.Choices) < 4 {
			return nil, errorx.Wrap(errors.New("50/50 is already used in this question"), errorx.Invalid)
		}

		//userGame.Lifeline.FiftyFifty = false
		isValid = true
		action = string(models.AssistanceTypeFiftyFifty)

		// update question
		correctAns := sessionUsing.CurrentQuestion.CorrectAnswer
		currentChoices := sessionUsing.CurrentQuestion.Choices

		leftKeyChoice := pkg.GenGoodRandom(0, 3, map[int]bool{correctAns: true})
		var newChoice []*models.Choice
		var choiceIndex []int
		for i, choice := range currentChoices {
			if choice.Key == leftKeyChoice || choice.Key == correctAns {
				newChoice = append(newChoice, choice)
				choiceIndex = append(choiceIndex, i)
			}
		}
		sessionUsing.CurrentQuestion.Choices = newChoice

		for _, translation := range sessionUsing.CurrentQuestion.Translations {
			var newChoice []*models.Choice
			for _, i := range choiceIndex {
				newChoice = append(newChoice, translation.Choices[i])
			}
			translation.Choices = newChoice
		}
	}

	if assistanceType == models.AssistanceTypeChangeQuestion && user.LifelineBalance > 1 {
		//userGame.Lifeline.ChangeQuestion = false
		action = string(models.AssistanceTypeChangeQuestion)

		questionSetup := &models.QuestionSetup{
			Difficulty: sessionUsing.CurrentQuestion.Difficulty,
		}

		// update question
		question, _, err := service.serviceQuestion.RandomNextQuestion(ctx, sessionUsing, questionSetup)
		if err != nil {
			return nil, err
		}

		sessionUsing.CurrentQuestion = question
		isValid = true
	}

	if !isValid {
		return nil, errorx.Wrap(errors.New("no assistance available"), errorx.NotExist)
	}

	err = service.useLifeline(ctx, user, action)
	if err != nil {
		return nil, err
	}

	session, err := redis_store.SaveGameSession(ctx, service.redisDB, sessionUsing)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (service *ServiceGame) canAnswer(ctx context.Context, user *models.User, userGame *models.UserGame, session *models.GameSession) error {
	serverMode, _ := service.serviceConfig.GetStringConfig(ctx, CONFIG_SERVER_MODE, SERVER_MODE_PRODUCTION)
	if serverMode == SERVER_MODE_STAGING {
		return nil
	}

	// TODO add constant
	if session.NextStep == 10 { // last question.
		return nil
	}

	return nil
}

func (service *ServiceGame) GetGames(ctx context.Context) ([]models.Game, error) {
	callback := func() ([]models.Game, error) {
		return datastore.GetEnabledGames(ctx, service.postgresDB)
	}

	games, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGames(), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return nil, err
	}
	return games, nil
}

func (service *ServiceGame) GetCurrentGameSession(ctx context.Context, userGame *models.UserGame) (*models.GameSession, error) {
	if userGame == nil {
		return nil, redis.Nil
	}
	// try to get current session from redis
	return redis_store.GetCurrentGameSessionByUser(ctx, service.redisDB, userGame.GameSlug, userGame.UserID)
}

// Deprecated
func (service *ServiceGame) GetGameSessionByID(ctx context.Context, userGame *models.UserGame, sessionID string) (*internal.GameSession, error) {
	if userGame == nil {
		return nil, errors.New("empty user game")
	}
	return redis_store.GetGameSessionByID(ctx, service.redisDB, userGame.GameSlug, userGame.UserID, sessionID)
}

func (service *ServiceGame) ReduceCountdown(ctx context.Context, user *models.User, gameSlug string) (bool, error) {
	mutex := service.rs.NewMutex(LockKeyUserBoost(user.ID))
	if err := mutex.Lock(); err != nil {
		return false, errorx.Wrap(ErrUserBoostLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	game, err := service.GetGame(ctx, gameSlug)
	if err != nil {
		return false, err
	}
	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if userGame == nil {
		return false, err
	}

	if userGame.Countdown == nil {
		return false, errorx.Wrap(errors.New("user countdown is nil"), errorx.Validation)
	}

	if time.Until(*userGame.Countdown) <= 0 {
		return false, errorx.Wrap(errors.New("user countdown is expired"), errorx.Validation)
	}

	//find current session
	session, err := service.GetCurrentGameSession(ctx, userGame)

	if err != nil {
		return false, err
	}

	numberBoostAllowed, err := service.GetGameIntConfig(ctx, gameSlug, NUMBER_OF_BOOST_ALLOWED, -1)
	if err != nil {
		log.Println("error while getting number of boost allowed", err)
	}

	if numberBoostAllowed > -1 {
		//update session current used boost +=1, if current used boost == 2, return error
		if session.UsedBoostCount >= numberBoostAllowed {
			return false, errorx.Wrap(errors.New("used boost count for this session is maxed out"), errorx.Validation)
		}
	}

	err = datastore.UseBoost(ctx, service.postgresDB, user.ID, models.ReduceTimeCountdown)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errorx.Wrap(errors.New("no boost available"), errorx.Validation)
		}
		return false, err
	}

	_ = service.cache.Delete(ctx, DBKeyMe(user.ID))
	_ = service.cache.Delete(ctx, DBKeyUser(user.ID))

	session.UsedBoostCount += 1
	_, err = redis_store.SaveGameSession(ctx, service.redisDB, session)
	if err != nil {
		return true, err
	}

	countdownTime := service.getReduceTimePerBoost(ctx, gameSlug)
	userGame, err = service.serviceUserGame.ReduceCountdown(ctx, userGame, countdownTime)

	return true, err
}

func (service *ServiceGame) GetGameSocialTask(ctx context.Context, gameSlug string) (*models.SocialTask, error) {
	callback := func() (*models.SocialTask, error) {
		return datastore.GetAvailableSocialTask(ctx, service.readonlyPostgresDB, gameSlug)
	}
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameSocialTasks(gameSlug), CACHE_TTL_15_MINS, callback)
}

func (service *ServiceGame) ConvertBoostToLifeline(ctx context.Context, user *models.User, gameSlug string) error {
	mutex := service.rs.NewMutex(LockKeyUserBoost(user.ID))
	if err := mutex.Lock(); err != nil {
		return errorx.Wrap(ErrUserBoostLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	game, err := service.GetGame(ctx, gameSlug)
	if err != nil {
		return err
	}
	userGame, err := service.serviceUserGame.GetUserGame(ctx, user, game)
	if userGame == nil {
		return err
	}

	err = datastore.UseBoost(ctx, service.postgresDB, user.ID, "convert-lifeline")
	if err != nil {
		return err
	}

	return service.serviceUser.ChangeLifelineBalance(ctx, user, "convert-lifeline", LIFELINES_PER_STAR)
}

func (service *ServiceGame) GetGameIntConfig(ctx context.Context, gameSlug string, key string, defaultValue int) (int, error) {
	callback := func() (int, error) {
		configs, err := datastore.GetGameConfig(ctx, service.readonlyPostgresDB, gameSlug)
		if err != nil {
			return defaultValue, err
		}

		for _, config := range configs {
			if config.Key == key {
				return strconv.Atoi(config.Value)
			}
		}

		return defaultValue, errorx.Wrap(errors.New("config not found"), errorx.NotExist)
	}

	value, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameConfig(gameSlug, key), CACHE_TTL_15_MINS, callback)
	if err != nil {
		return defaultValue, err
	}

	return value, nil
}

func (service *ServiceGame) GetGameStringConfig(ctx context.Context, gameSlug string, key string, defaultValue string) (string, error) {
	callback := func() (string, error) {
		configs, err := datastore.GetGameConfig(ctx, service.readonlyPostgresDB, gameSlug)
		if err != nil {
			return defaultValue, err
		}

		for _, config := range configs {
			if config.Key == key {
				return config.Value, nil
			}
		}

		return defaultValue, errorx.Wrap(errors.New("config not found"), errorx.NotExist)
	}

	value, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameConfig(gameSlug, key), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return defaultValue, err
	}

	return value, nil
}

func (service *ServiceGame) getUserSocialBonusScore(ctx context.Context, user *models.User) (int, error) {
	callback := func() (int, error) {
		tasks, err := service.serviceSocial.GetAvailableSocialTasksByUser(ctx, user.ID)
		if err != nil {
			return 0, err
		}

		bonusPoint := 0
		for _, task := range tasks {
			for _, link := range task.Links {
				if link.Joined == true {
					bonusPoint += link.Gem
				}
			}
		}

		return bonusPoint, nil
	}

	point, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserSocialBonusScore(user.ID), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return 0, err
	}

	return point, nil
}

func (service *ServiceGame) getGameCountdownTime(ctx context.Context, gameSlug string) time.Duration {
	callback := func() (time.Duration, error) {
		timeCountdownInt, _ := service.GetGameIntConfig(ctx, gameSlug, TIME_COUNTDOWN, DEFAULT_SESSION_COUNTDOWN_IN_MINUTES)
		timeCountdown := time.Duration(timeCountdownInt) * time.Minute
		return timeCountdown, nil
	}

	countdownTime, _ := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameCountdownTime(gameSlug), CACHE_TTL_15_MINS, callback)
	return countdownTime
}

func (service *ServiceGame) getReduceTimePerBoost(ctx context.Context, gameSlug string) time.Duration {
	callback := func() (time.Duration, error) {
		reduceTime, _ := service.GetGameIntConfig(ctx, gameSlug, TIME_REDUCE_PER_BOOST, DEFAULT_TIME_REDUCE_PER_BOOST_IN_MINUTES)
		return time.Duration(reduceTime) * time.Minute, nil
	}

	reduceTime, _ := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameReduceTimePerBoost(gameSlug), CACHE_TTL_15_MINS, callback)
	return reduceTime
}

//func (service *ServiceGame) getLifelineHistory(ctx context.Context, userID string) ([]models.LifelineHistory, error) {
//}

func (service *ServiceGame) useLifeline(ctx context.Context, user *models.User, action string) error {
	return service.serviceUser.ChangeLifelineBalance(ctx, user, action, -1)
}
