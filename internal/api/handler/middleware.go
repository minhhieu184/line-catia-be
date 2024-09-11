package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"millionaire/internal/models"
	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type ctxKey string

var ctxKeyAuthUser ctxKey = "AUTH_USER"
var ctxKeyAuthPartner ctxKey = "AUTH_PARTNER"

func Authn(verifier interface {
	ValidateInitData(dataStr string) (*models.UserFromAuth, error)
},
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return next(c)
			}

			parts := strings.Split(header, "Bearer")
			if len(parts) != 2 {
				return next(c)
			}

			token := strings.TrimSpace(parts[1])
			if len(token) == 0 {
				return next(c)
			}

			println("token", token)

			user, err := verifier.ValidateInitData(token)
			if err != nil {
				// although it's a client error, we don't want to detailed information
				//nolint:errcheck
				httpx.Abort(c, errorx.Wrap(errors.New("invalid access token"), errorx.Authn), -1)
				return nil
			}

			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ctxKeyAuthUser, user)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func ResolveValidUser(ctx context.Context, container *do.Injector) (*models.User, error) {
	userAuth, ok := ctx.Value(ctxKeyAuthUser).(*models.UserFromAuth)
	if !ok {
		return nil, errorx.Wrap(errors.New("missing session"), errorx.Authn)
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](container)
	if err != nil {
		return nil, err
	}

	return serviceUser.FindOrCreateUser(ctx, userAuth)
}

func middlewareTimeEndedGameContext(container *do.Injector) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			gameSlug := c.Param("game")
			if gameSlug == "" {
				return next(c)
			}

			serviceGame, err := do.Invoke[*services.ServiceGame](container)
			if err != nil {
				return err
			}

			game, err := serviceGame.GetGame(c.Request().Context(), gameSlug)
			if err != nil {
				return err
			}

			if game.StartTime != nil && time.Now().Before(*game.StartTime) {
				httpx.Abort(c, errorx.Wrap(errors.New("game not started yet"), errorx.Other), -1)
				return nil
			}

			if game.EndTime != nil && time.Now().After(*game.EndTime) {
				httpx.Abort(c, errorx.Wrap(errors.New("game ended"), errorx.Other), -1)
				return nil
			}

			return next(c)
		}
	}
}

// middleware function to check valid partner via api key header, save partner to context
func AuthnPartner(verifier interface {
	ValidateAPIKey(apiKey string) (*models.Partner, error)
},
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("X-Api-Key")
			if header == "" {
				httpx.Abort(c, errorx.Wrap(errors.New("unauthorized"), errorx.Authn), -1)
				return nil
			}

			partner, err := verifier.ValidateAPIKey(header)
			if err != nil {
				httpx.Abort(c, errorx.Wrap(errors.New("unauthorized"), errorx.Authn), -1)
				return nil
			}

			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ctxKeyAuthPartner, partner)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func ResolveValidPartner(ctx context.Context, container *do.Injector) (*models.Partner, error) {
	userAuth, ok := ctx.Value(ctxKeyAuthPartner).(*models.Partner)
	if !ok {
		return nil, errorx.Wrap(errors.New("missing session"), errorx.Authn)
	}

	return userAuth, nil
}
