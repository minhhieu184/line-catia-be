package handler

import (
	"millionaire/internal/models"
	"millionaire/internal/services"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

type groupUser struct {
	container *do.Injector
}

func (gr *groupUser) Me(c echo.Context) error {
	ctx := c.Request().Context()

	// find user in system. If not create new user
	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	refCodeParam := c.QueryParam("refCode")
	user, err = serviceUser.Me(ctx, user, refCodeParam)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	claims := &services.CustomClaims{
		ID:       user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}, nil)
}

func (gr *groupUser) ClaimUserBoost(c echo.Context) error {
	ctx := c.Request().Context()

	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 0
	limit := 10
	//if page or limit is empty, set default value
	if limitStr == "" {
		limit = 10
	} else {
		limit, _ = strconv.Atoi(limitStr)

		if limit > 100 {
			limit = 100
		}
	}

	page, _ = strconv.Atoi(pageStr)
	if page < 0 {
		page = 0
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	source := c.Param("source")
	err = serviceUser.ClaimUserBoost(ctx, source, user.ID, page, limit)

	return httpx.RestAbort(c, nil, err)
}

func (gr *groupUser) GetFriendList(c echo.Context) error {
	ctx := c.Request().Context()

	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 0
	limit := 10
	//if page or limit is empty, set default value
	if limitStr == "" {
		limit = 10
	} else {
		limit, _ = strconv.Atoi(limitStr)

		if limit > 100 {
			limit = 100
		}
	}

	page, _ = strconv.Atoi(pageStr)
	if page < 0 {
		page = 0
	}

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	friendList, err := serviceUser.GetUserFriendListPaging(ctx, user.ID, page, limit)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	friendCount, err := serviceUser.CountUserFriends(ctx, user.ID)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	return httpx.RestAbort(c, map[string]interface{}{
		"friend_list":  friendList,
		"friend_count": friendCount,
	}, nil)
}

func (gr *groupUser) ClaimAllBoosts(c echo.Context) error {
	ctx := c.Request().Context()

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	friends, err := serviceUser.ClaimAllAvailableBoostFromFriends(ctx, user)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, friends, nil)
}

func (gr *groupUser) ConnectTonWallet(c echo.Context) error {
	ctx := c.Request().Context()

	user, err := ResolveValidUser(c.Request().Context(), gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	var payload models.TonProof
	if err := c.Bind(&payload); err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Invalid))
	}

	serviceUser, err := do.Invoke[*services.ServiceUser](gr.container)
	if err != nil {
		return httpx.RestAbort(c, nil, errorx.Wrap(err, errorx.Service))
	}

	err = serviceUser.ConnectTonWallet(ctx, user, &payload)

	if err != nil {
		return httpx.RestAbort(c, nil, err)
	}

	return httpx.RestAbort(c, "success", nil)
}
