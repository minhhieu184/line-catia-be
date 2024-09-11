package handler

import (
	"errors"
	"millionaire/internal/services"
	"strings"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupLeaderboard struct {
	container *do.Injector
}

func (gr *groupLeaderboard) GetTopReferralLeaderboard(c echo.Context) error {
	serviceLeaderboard, err := do.Invoke[*services.ServiceLeaderboard](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()

	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	leaderboard, err := serviceLeaderboard.GetTopReferralLeaderboard(ctx, user)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	return httpx.RestAbort(c, leaderboard, nil)
}

func (gr *groupLeaderboard) GetOverallLeaderboard(c echo.Context) error {
	serviceLeaderboard, err := do.Invoke[*services.ServiceLeaderboard](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()

	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	leaderboard, err := serviceLeaderboard.GetOverallLeaderboard(ctx, user)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	return httpx.RestAbort(c, leaderboard, nil)
}

func (gr *groupLeaderboard) GetWeeklyOverallLeaderboard(c echo.Context) error {
	serviceLeaderboard, err := do.Invoke[*services.ServiceLeaderboard](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()

	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	leaderboard, err := serviceLeaderboard.GetWeeklyOverallLeaderboard(ctx, user)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	return httpx.RestAbort(c, leaderboard, nil)
}

func (gr *groupLeaderboard) GetGameLeaderboard(c echo.Context) error {
	serviceLeaderboard, err := do.Invoke[*services.ServiceLeaderboard](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	game := c.Param("game")
	if game == "" || game == "undefined" {
		return httpx.RestAbort(c, nil, errorx.Wrap(errors.New("game is required"), errorx.Invalid))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	gameLeaderboard, err := serviceLeaderboard.GetGameLeaderboard(ctx, strings.ToLower(game), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, gameLeaderboard, nil)
}
