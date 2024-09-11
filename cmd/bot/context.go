package main

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	tele "gopkg.in/telebot.v3"
)

func getContextPostgres(context tele.Context) (*bun.DB, error) {
	contextValue := context.Get(contextPostgres)
	if contextValue == nil {
		return nil, fmt.Errorf("database not found")
	}

	result, ok := contextValue.(*bun.DB)
	if !ok {
		return nil, fmt.Errorf("database not valid")
	}

	return result, nil
}

func getContextRedis(context tele.Context) (redis.UniversalClient, error) {
	contextValue := context.Get(contextRedis)
	if contextValue == nil {
		return nil, fmt.Errorf("cache not found")
	}

	result, ok := contextValue.(redis.UniversalClient)
	if !ok {
		return nil, fmt.Errorf("cache not valid")
	}

	return result, nil
}

func getContextRedisCache(context tele.Context) (redis.UniversalClient, error) {
	contextValue := context.Get(contextRedisCache)
	if contextValue == nil {
		return nil, fmt.Errorf("cache not found")
	}

	result, ok := contextValue.(redis.UniversalClient)
	if !ok {
		return nil, fmt.Errorf("cache not valid")
	}

	return result, nil
}
