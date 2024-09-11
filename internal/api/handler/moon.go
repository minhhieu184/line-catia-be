package handler

import (
	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupMoon struct {
	container *do.Injector
}

func (gr *groupMoon) GetMoon(c echo.Context) error {
	serviceMoon, err := do.Invoke[*services.ServiceMoon](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	moon, err := serviceMoon.GetUserMoon(c.Request().Context(), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, moon, nil)
}

func (gr *groupMoon) SpinGacha(c echo.Context) error {
	serviceMoon, err := do.Invoke[*services.ServiceMoon](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	moon, err := serviceMoon.SpinGacha(c.Request().Context(), user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, moon, nil)
}
