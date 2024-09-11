package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"millionaire/internal/api/handler"
	"millionaire/internal/interfaces"
	"millionaire/internal/pkg/caching"
	"millionaire/internal/pkg/limiter"
	"millionaire/internal/services"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/hiendaovinh/toolkit/pkg/env"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func init() {
	// for development
	//nolint:errcheck
	godotenv.Load("../../.env")

	// for production
	//nolint:errcheck
	godotenv.Load("./.env")
}

func main() {
	vs, err := env.EnvsRequired(
		"BOT_TOKEN",
		"JWT_SECRET",
		"CHANCES",
		"DB_DSN",
		"TON_APP_DOMAIN",
	)
	if err != nil {
		log.Fatal(err)
	}

	container := NewContainer(vs)

	app := &cli.App{
		Name: "api",
		Commands: []*cli.Command{
			commandServer(container),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandServer(container *do.Injector) *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "start the web server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Value: "0.0.0.0:8080",
				Usage: "serve address",
			},
		},
		Action: func(c *cli.Context) error {
			vs := do.MustInvokeNamed[map[string]string](container, "envs")
			router, err := handler.New(&handler.Config{
				Container: container,
				Mode:      vs["API_MODE"],
				Origins:   strings.Split(vs["API_ORIGINS"], ","),
			})
			if err != nil {
				fmt.Println("111")
				fmt.Println(err)
				return err
			}

			srv := &http.Server{
				Addr:    c.String("addr"),
				Handler: router,
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errWg, errCtx := errgroup.WithContext(ctx)

			errWg.Go(func() error {
				log.Printf("ListenAndServe: %s (%s)\n", c.String("addr"), vs["API_MODE"])
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					fmt.Println("222")
					fmt.Println(err)
					return err
				}
				return nil
			})

			errWg.Go(func() error {
				<-errCtx.Done()
				return srv.Shutdown(context.TODO())
			})

			return errWg.Wait()
		},
	}
}

func NewContainer(vs map[string]string) *do.Injector {
	injector := do.New()
	vs["CHANCES"] = os.Getenv("CHANCES")
	vs["API_MODE"] = os.Getenv("API_MODE")
	vs["API_ORIGINS"] = os.Getenv("API_ORIGINS")
	vs["TON_APP_DOMAIN"] = os.Getenv("TON_APP_DOMAIN")

	if vs["API_MODE"] == "" {
		vs["API_MODE"] = "production"
	}
	if vs["API_ORIGINS"] == "" {
		vs["API_ORIGINS"] = "*"
	}

	do.ProvideNamedValue(injector, "envs", vs)

	do.Provide(injector, func(i *do.Injector) (*bun.DB, error) {
		godotenv.Load()
		sqldb := sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithDSN(os.Getenv("DB_DSN")),
			pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
		))

		db := bun.NewDB(sqldb, pgdialect.New())
		return db, nil
	})

	do.ProvideNamed(injector, "db-readonly", func(i *do.Injector) (*bun.DB, error) {
		godotenv.Load()
		sqldb := sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithDSN(os.Getenv("DB_DSN_READONLY")),
			pgdriver.WithPassword(os.Getenv("DB_PASSWORD_READONLY")),
		))

		db := bun.NewDB(sqldb, pgdialect.New())
		return db, nil
	})

	do.ProvideNamed(injector, "redis-db", func(i *do.Injector) (redis.UniversalClient, error) {
		clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_QUESTIONNAIRE")
		if clusterCacheRedisURL != "" {
			clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
			if err != nil {
				return nil, err
			}
			return redis.NewClusterClient(clusterOpts), nil
		}
		return db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_QUESTIONNAIRE"),
		})
	})

	do.ProvideNamed(injector, "redis-cache", func(i *do.Injector) (redis.UniversalClient, error) {
		clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_CACHE")
		if clusterCacheRedisURL != "" {
			clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
			if err != nil {
				return nil, err
			}
			return redis.NewClusterClient(clusterOpts), nil
		}
		return db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_CACHE"),
		})
	})

	do.ProvideNamed(injector, "redis-cache-readonly", func(i *do.Injector) (redis.UniversalClient, error) {
		var clusterOpts *redis.ClusterOptions
		var err error
		clusterCacheRedisReadOnlyURL := os.Getenv("CLUSTER_REDIS_CACHE_READONLY")
		if clusterCacheRedisReadOnlyURL != "" {
			clusterOpts, err = redis.ParseClusterURL(clusterCacheRedisReadOnlyURL)
		} else {
			clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_CACHE")
			if clusterCacheRedisURL != "" {
				clusterOpts, err = redis.ParseClusterURL(clusterCacheRedisURL)
			}
		}

		if err != nil {
			return nil, err
		}
		if clusterOpts != nil {
			clusterOpts.ReadOnly = true
			return redis.NewClusterClient(clusterOpts), nil
		}

		return db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_CACHE_READONLY"),
		})
	})

	do.ProvideNamed(injector, "redis-limiter", func(i *do.Injector) (redis.UniversalClient, error) {
		clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_LIMITER")
		if clusterCacheRedisURL != "" {
			clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
			if err != nil {
				return nil, err
			}
			return redis.NewClusterClient(clusterOpts), nil
		}

		a, e := db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_LIMITER"),
		})
		return a, e
	})

	do.ProvideNamed(injector, "redis-mutex", func(i *do.Injector) (redis.UniversalClient, error) {
		clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_MUTEX")
		if clusterCacheRedisURL != "" {
			clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
			if err != nil {
				return nil, err
			}
			return redis.NewClusterClient(clusterOpts), nil
		}

		a, e := db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_MUTEX"),
		})
		return a, e
	})

	do.Provide(injector, func(i *do.Injector) (caching.Cache, error) {
		dbRedis, err := do.InvokeNamed[redis.UniversalClient](i, "redis-cache")
		if err != nil {
			return nil, err
		}

		return caching.NewCacheRedis(dbRedis, false)
	})

	do.Provide(injector, func(i *do.Injector) (caching.ReadOnlyCache, error) {
		dbRedis, err := do.InvokeNamed[redis.UniversalClient](i, "redis-cache-readonly")
		if err != nil {
			return nil, err
		}

		return caching.NewCacheRedis(dbRedis, false)
	})

	do.Provide(injector, func(i *do.Injector) (interfaces.Limiter, error) {
		dbRedis, err := do.InvokeNamed[redis.UniversalClient](i, "redis-limiter")
		if err != nil {
			return nil, err
		}

		a, err := limiter.NewLimiter(dbRedis)
		return a, err
	})

	do.Provide(injector, func(i *do.Injector) (*redsync.Redsync, error) {
		dbRedis, err := do.InvokeNamed[redis.UniversalClient](i, "redis-mutex")
		if err != nil {
			return nil, err
		}

		pool := goredis.NewPool(dbRedis)
		rs := redsync.New(pool)
		return rs, nil
	})

	do.Provide(injector, func(i *do.Injector) (*services.Bot, error) {
		return services.NewBot(vs["BOT_TOKEN"])
	})

	do.Provide(injector, func(i *do.Injector) (*services.Authentication, error) {
		return services.NewAuthentication(vs["JWT_SECRET"])
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceGame, error) {
		return services.NewServiceGame(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceUser, error) {
		return services.NewServiceUser(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceSocial, error) {
		return services.NewServiceSocial(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceUserGame, error) {
		return services.NewServiceUserGame(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceQuestion, error) {
		return services.NewServiceQuestion(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceConfig, error) {
		return services.NewServiceConfig(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceLeaderboard, error) {
		return services.NewServiceLeaderboard(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceUserFreebies, error) {
		return services.NewServiceUserFreebies(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServicePartner, error) {
		return services.NewServicePartner(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceMoon, error) {
		return services.NewServiceMoon(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceReward, error) {
		return services.NewServiceReward(injector)
	})

	do.Provide(injector, func(i *do.Injector) (*services.ServiceArena, error) {
		return services.NewServiceArena(injector)
	})

	return injector
}
