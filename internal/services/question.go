package services

import (
	"context"
	"errors"
	"math/rand"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"strconv"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceQuestion struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache
}

func NewServiceQuestion(container *do.Injector) (*ServiceQuestion, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	rs, err := do.Invoke[*redsync.Redsync](container)
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

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	return &ServiceQuestion{container, db, rs, postgresDB, readonlyPostgresDB, cache, readonlyCache}, nil
}

func (service *ServiceQuestion) GetQuestion(ctx context.Context, questionID int) (*models.Question, error) {
	callback := func() (*models.Question, error) {
		question, err := datastore.GetQuestion(ctx, service.readonlyPostgresDB, questionID)
		if err != nil {
			return nil, err
		}

		if question != nil {
			question.Translations, _ = datastore.GetQuestionTranslation(ctx, service.readonlyPostgresDB, question.QuestionBankID, question.Category)
		}

		return question, nil
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyQuestion(questionID), CACHE_TTL_15_MINS, callback)
}

func (service *ServiceQuestion) RandomNextQuestion(ctx context.Context, session *models.GameSession, questionSetup *models.QuestionSetup) (*models.Question, int, error) {
	if session == nil {
		return nil, 0, errorx.Wrap(errors.New("session not found"), errorx.NotExist)
	}

	err := service.checkQuestionGroupExistence(ctx, session, questionSetup)

	if err != nil {
		return nil, 0, err
	}

	mapQuestionUsed := make(map[int]bool)
	for _, questionHistory := range session.History {
		quesID := questionHistory.Question.ID
		mapQuestionUsed[quesID] = true
	}

	var question *models.Question
	count := 0
	for {
		count++
		if count > 100 {
			break
		}
		questionId := redis_store.RandomQuestionFromGroup(ctx, service.redisDB, session.GameSlug, string(questionSetup.Difficulty))
		id, err := strconv.Atoi(questionId)
		if err != nil {
			continue
		}

		if !mapQuestionUsed[id] {
			question, _ = service.GetQuestion(ctx, id)
			if question != nil {
				break
			}
		}
	}

	if question == nil {
		return nil, 0, errorx.Wrap(errors.New("no question available"), errorx.NotExist)
	}

	randomChoicesQuestion := question.Choices
	rand.NewSource(time.Now().UnixNano())
	rand.Shuffle(len(randomChoicesQuestion), func(i, j int) {
		//loop question translation
		for _, translation := range question.Translations {
			translation.Choices[i], translation.Choices[j] = translation.Choices[j], translation.Choices[i]
		}

		randomChoicesQuestion[i], randomChoicesQuestion[j] = randomChoicesQuestion[j], randomChoicesQuestion[i]

	})

	return question, questionSetup.Score, nil
}

func (service *ServiceQuestion) checkQuestionGroupExistence(ctx context.Context, session *models.GameSession, questionSetup *models.QuestionSetup) error {
	//check if dbKeyQuestionGroup exist
	q, err := redis_store.GetQuestionGroup(ctx, service.redisDB, session.GameSlug, string(questionSetup.Difficulty))
	if err == redis.Nil || len(q) == 0 {
		//get question from db and set to redis
		callback := func() ([]*models.GameCategory, error) {
			return datastore.GetGameCategory(ctx, service.readonlyPostgresDB, session.GameSlug)
		}

		gameCategories, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameCategory(session.GameSlug), CACHE_TTL_1_HOUR, callback)
		if err != nil {
			return err
		}

		questions, err := datastore.GetQuestionIdsAndDifficulty(ctx, service.readonlyPostgresDB, gameCategories)
		if err != nil {
			return err
		}
		questionGroups := make(map[string][]int)

		for _, question := range questions {
			group := questionGroups[string(question.Difficulty)]
			if group == nil {
				group = make([]int, 0)

			}
			group = append(group, question.QuestionId)
			questionGroups[string(question.Difficulty)] = group
		}

		for key, group := range questionGroups {
			err = redis_store.AddQuestionsToGroup(ctx, service.redisDB, session.GameSlug, key, group)
		}

		return err
	}

	return nil
}
