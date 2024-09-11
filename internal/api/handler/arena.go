package handler

import (
	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupArena struct {
	container *do.Injector
}

func (gr *groupArena) GetArenas(c echo.Context) error {
	serviceArena, err := do.Invoke[*services.ServiceArena](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	arenas, err := serviceArena.GetEnabledArenas(c.Request().Context())
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, arenas, nil)
}

func (gr *groupArena) GetArena(c echo.Context) error {
	serviceArena, err := do.Invoke[*services.ServiceArena](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceSocial, err := do.Invoke[*services.ServiceSocial](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	slug := c.Param("slug")
	arena, err := serviceArena.GetArena(c.Request().Context(), slug)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	game, err := serviceGame.GetGame(c.Request().Context(), arena.GameSlug)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	tasks, err := serviceSocial.GetUserTasks(c.Request().Context(), user.ID, game.Slug)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(c.Request().Context(), user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, map[string]interface{}{
		"arena":     arena,
		"game":      game,
		"tasks":     tasks,
		"user_game": userGame,
	}, nil)
}

func (gr *groupArena) GetArenaLeaderboard(c echo.Context) error {
	serviceLeaderboard, err := do.Invoke[*services.ServiceLeaderboard](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	ctx := c.Request().Context()

	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	slug := c.Param("slug")
	leaderboard, err := serviceLeaderboard.GetArenaLeaderboard(ctx, slug, user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, leaderboard, nil)
}
