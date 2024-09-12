package handler

import (
	"errors"
	"millionaire/internal/models"
	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupGame struct {
	container *do.Injector
}

func (gr *groupGame) Show(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	socialTasks, _ := serviceGame.GetGameSocialTask(ctx, game.Slug)

	gameInfo := struct {
		*models.Game
		Task *models.SocialTask `json:"task"`
	}{
		Game: game,
		Task: socialTasks,
	}

	return httpx.RestAbort(c, gameInfo, nil)
}

func (gr *groupGame) FindOrCreateSession(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}
	println("user id xcv", user.ID)

	session, err := serviceGame.FindOrCreateSession(ctx, c.Param("game"), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) CurrentSession(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.GetCurrentGameSession(ctx, userGame)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) Next(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.NextQuestion(ctx, c.Param("game"), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) Answer(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	var payload models.GameAnswer
	if err := c.Bind(&payload); err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Invalid))
	}

	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.Answer(ctx, user, userGame, payload)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) End(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.QuitGame(ctx, c.Param("game"), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) Scores(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.GetUserGameScore(ctx, userGame, user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) GetLastUserSessionScore(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	session, err := serviceGame.GetLastUserSessionScore(ctx, userGame, user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) BurnAssistance(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	var payload AssistancePayload
	if err := c.Bind(&payload); err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Invalid))
	}

	if !payload.AssistanceType.Valid() {
		return httpx.RestAbort(c, nil, errorx.Wrap(errors.New("invalid assistance"), errorx.Invalid))
	}

	session, err := serviceGame.BurnAssistance(ctx, c.Param("game"), payload.AssistanceType, user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, session, nil)
}

func (gr *groupGame) GetGames(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	games, err := serviceGame.GetGames(ctx)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, games, nil)
}

func (gr *groupGame) Me(c echo.Context) error {
	ctx := c.Request().Context()

	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	// find user in system. If not create new user
	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	game, err := serviceGame.GetGame(ctx, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGame, err := serviceUserGame.GetUserGame(ctx, user, game)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, userGame, nil)
}

func (gr *groupGame) ReduceCountdown(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	result, err := serviceGame.ReduceCountdown(ctx, user, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, result, nil)
}

func (gr *groupGame) ConvertBoostToLifeline(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	err = serviceGame.ConvertBoostToLifeline(ctx, user, c.Param("game"))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, true, nil)
}

func (gr *groupGame) GetUserGameList(c echo.Context) error {
	serviceGame, err := do.Invoke[*services.ServiceGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceUserGame, err := do.Invoke[*services.ServiceUserGame](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	ctx := c.Request().Context()
	user, err := ResolveValidUser(ctx, gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	games, err := serviceGame.GetGames(ctx)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userGameList, err := serviceUserGame.GetUserGameList(ctx, user, games)

	return httpx.RestAbort(c, userGameList, nil)
}

type AssistancePayload struct {
	AssistanceType models.AssistanceType `json:"assistance_type"`
}
