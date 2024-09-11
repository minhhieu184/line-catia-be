package handler

import (
	"context"
	"millionaire/internal/services"
	"strconv"

	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupSocial struct {
	container *do.Injector
}

func (gr *groupSocial) GetTasks(c echo.Context) error {
	serviceSocial, err := do.Invoke[*services.ServiceSocial](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	gameSlug := c.Param("game")

	tasks, err := serviceSocial.GetUserTasks(c.Request().Context(), user.ID, gameSlug)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, tasks, nil)
}

func (gr *groupSocial) VerifyTask(c echo.Context) error {
	serviceSocial, err := do.Invoke[*services.ServiceSocial](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	linkIdString := c.Param("link-id")
	linkId, err := strconv.Atoi(linkIdString)

	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	gameSlug := c.Param("game")
	ctx := context.Background()

	task, err := serviceSocial.VerifySocialTask(ctx, user, gameSlug, linkId)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, task, nil)
}

func (gr *groupSocial) GetAvailableSocialTasks(c echo.Context) error {
	serviceSocial, err := do.Invoke[*services.ServiceSocial](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	tasks, err := serviceSocial.GetAvailableSocialTasksByUser(c.Request().Context(), user.ID)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, tasks, nil)
}
