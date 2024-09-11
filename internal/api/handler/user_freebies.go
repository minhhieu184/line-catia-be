package handler

import (
	"log"
	"millionaire/internal/models"
	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupUserFreebies struct {
	container *do.Injector
}

func (gr *groupUserFreebies) GetUserFreebies(c echo.Context) error {
	ctx := c.Request().Context()

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUserFreebies, err := do.Invoke[*services.ServiceUserFreebies](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	userFreebies, err := serviceUserFreebies.GetOrNewUserFreebies(ctx, user.ID)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	serviceConfig, err := do.Invoke[*services.ServiceConfig](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	for _, userFreebie := range userFreebies {
		if userFreebie.Action == models.ACTION_CLAIM_GEM {
			timeConfig, _ := serviceConfig.GetIntConfig(ctx, services.CONFIG_FREEBIE_GEM_COUNTDOWN, 5)
			if userFreebie.Amount == 0 {
				userFreebie.Amount = services.GEM_AMOUNT
			}

			userFreebie.ClaimSchedule = timeConfig
		}

		if userFreebie.Action == models.ACTION_CLAIM_LIFELINE {
			timeConfig, _ := serviceConfig.GetIntConfig(ctx, services.CONFIG_FREEBIE_LIFELINE_COUNTDOWN, 5)
			if userFreebie.Amount == 0 {
				userFreebie.Amount = services.LIFELINE_AMOUNT
			}

			userFreebie.ClaimSchedule = timeConfig
		}

		if userFreebie.Action == models.ACTION_CLAIM_STAR {
			timeConfig, _ := serviceConfig.GetIntConfig(ctx, services.CONFIG_FREEBIE_STAR_COUNTDOWN, 5)
			if userFreebie.Amount == 0 {
				userFreebie.Amount = services.STAR_AMOUNT
			}

			userFreebie.ClaimSchedule = timeConfig
		}
	}

	latestMessage, err := serviceUserFreebies.GetLatestMessage(ctx)
	if err != nil {
		log.Println(err)
	}

	return httpx.RestAbort(c, map[string]interface{}{
		"freebies": userFreebies,
		"message":  latestMessage,
	}, nil)
}

func (gr *groupUserFreebies) ClaimFreebies(c echo.Context) error {
	ctx := c.Request().Context()

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUserFreebies, err := do.Invoke[*services.ServiceUserFreebies](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	err = serviceUserFreebies.ClaimFreebies(ctx, user, c.Param("action"))
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	return httpx.RestAbort(c, nil, nil)
}
