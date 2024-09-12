package handler

import (
	"millionaire/internal/services"
	"strconv"

	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupPartner struct {
	container *do.Injector
}

func (gr *groupPartner) CheckUserJoined(c echo.Context) error {
	//get apikey from x-header
	partner, err := ResolveValidPartner(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	userIdStr := c.QueryParam("user_id")
	// userId, _ := strconv.ParseInt(userIdStr, 10, 64)
	userId := userIdStr

	servicePartner, err := do.Invoke[*services.ServicePartner](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	refCodeStr := c.QueryParam("ref_code")
	refCode := "-1"
	if refCodeStr != "" {
		// refCode, _ = strconv.ParseInt(refCodeStr, 10, 64)
		refCode = refCodeStr
	}

	minGemStr := c.QueryParam("min_gem")
	minGem := -1
	if minGemStr != "" {
		minGem, _ = strconv.Atoi(minGemStr)
	}

	response, err := servicePartner.CheckJoinedUser(c.Request().Context(), partner, userId, refCode, minGem)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, response, nil)
}

type CustomResponse struct {
	User bool `json:"user"`
	Gem  bool `json:"gem"`
	Ref  bool `json:"code"`
}
