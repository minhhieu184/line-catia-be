package handler

import (
	"net/http"

	"millionaire/internal/services"

	"github.com/hiendaovinh/toolkit/pkg/httpx-echo"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/do"
)

type Config struct {
	Container *do.Injector
	Mode      string
	Origins   []string
}

func New(cfg *Config) (http.Handler, error) {
	r := echo.New()
	r.Pre(middleware.RemoveTrailingSlash())
	if cfg.Mode == "debug" {
		r.Debug = true
		pprof.Register(r)
	}

	r.JSONSerializer = httpx.SegmentJSONSerializer{}
	r.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339}\t${method}\t${uri}\t${status}\t${latency_human}\n",
	}))
	r.Use(middleware.Recover())

	r.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "🤖")
	})

	routesAPIv1 := r.Group("/api/v1")
	{
		bot, err := do.Invoke[*services.Bot](cfg.Container)
		if err != nil {
			return nil, err
		}
		authentication, err := do.Invoke[*services.Authentication](cfg.Container)
		if err != nil {
			return nil, err
		}
		cors := middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     cfg.Origins,
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
			MaxAge:           60 * 60,
		})

		routesAPIv1.Use(cors)

		routesAPIv1Me := routesAPIv1.Group("/user/me")
		routesAPIv1Me.Use(Authn(bot))
		{
			m := groupUser{cfg.Container}
			routesAPIv1Me.GET("", m.Me)
		}

		routesAPIv1.Use(Authn(authentication)) // Authn will NOT terminate unauthenticated request.
		routesAPIv1.GET("", Hello)

		routesAPIv1User := routesAPIv1.Group("/user")
		{
			u := groupUser{cfg.Container}
			// routesAPIv1User.GET("/me", u.Me)
			routesAPIv1User.POST("/boost/claim/:source", u.ClaimUserBoost)
			routesAPIv1User.GET("/friends", u.GetFriendList)
			routesAPIv1User.POST("/boost/claim-all", u.ClaimAllBoosts)
			routesAPIv1User.POST("/connect/ton", u.ConnectTonWallet)
		}

		g := groupGame{cfg.Container}
		routesAPIv1.GET("/games", g.GetGames)
		// no need to check game end time here
		routesAPIv1.GET("/game/:game", g.Show)

		routesAPIv1.GET("/game/:game/scores", g.Scores)

		l := groupLeaderboard{cfg.Container}
		routesAPIv1.GET("/leaderboard/referral", l.GetTopReferralLeaderboard)
		routesAPIv1.GET("/leaderboard/overall", l.GetOverallLeaderboard)
		routesAPIv1.GET("/leaderboard/overall_weekly", l.GetWeeklyOverallLeaderboard)
		routesAPIv1.GET("/game/:game/leaderboard", l.GetGameLeaderboard)

		routesAPIv1Game := routesAPIv1.Group("/game")
		{
			routesAPIv1Game.Use(middlewareTimeEndedGameContext(cfg.Container))
			g := groupGame{cfg.Container}

			routesAPIv1Game.GET("/:game/session", g.FindOrCreateSession)
			routesAPIv1Game.GET("/:game/current_session", g.FindOrCreateSession)
			routesAPIv1Game.GET("/:game/next", g.Next)
			routesAPIv1Game.GET("/:game/me", g.Me)
			routesAPIv1Game.POST("/:game/current_session/end", g.End)
			routesAPIv1Game.POST("/:game/current_session/answer", g.Answer)
			routesAPIv1Game.POST("/:game/current_session/assistance", g.BurnAssistance)
			routesAPIv1Game.POST("/:game/reduce-countdown", g.ReduceCountdown)
			routesAPIv1Game.POST("/:game/convert-lifeline", g.ConvertBoostToLifeline)
			routesAPIv1Game.GET("/:game/last_session_score", g.GetLastUserSessionScore)
			routesAPIv1Game.GET("/game-list", g.GetUserGameList)
		}

		s := groupSocial{cfg.Container}
		routesAPIv1.GET("/socials/tasks/:game", s.GetTasks)
		routesAPIv1.GET("/socials/join/:game/:link-id", s.VerifyTask)
		routesAPIv1.GET("/socials/tasks", s.GetAvailableSocialTasks)

		uf := groupUserFreebies{cfg.Container}
		routesAPIv1.GET("/user/freebies", uf.GetUserFreebies)
		routesAPIv1.POST("/user/freebies/claim/:action", uf.ClaimFreebies)

		routesAPIv1Parter := routesAPIv1.Group("/3rd")

		{
			partner, err := do.Invoke[*services.ServicePartner](cfg.Container)
			if err != nil {
				return nil, err
			}

			routesAPIv1Parter.Use(AuthnPartner(partner))
			p := groupPartner{cfg.Container}
			routesAPIv1Parter.GET("/verify-user", p.CheckUserJoined)
		}

		m := groupMoon{cfg.Container}
		routesAPIv1.GET("/moon", m.GetMoon)
		routesAPIv1.POST("/moon/spin", m.SpinGacha)

		a := groupArena{cfg.Container}
		routesAPIv1.GET("/arenas", a.GetArenas)
		routesAPIv1.GET("/arena/:slug", a.GetArena)
		routesAPIv1.GET("/arena/:slug/leaderboard", a.GetArenaLeaderboard)
	}

	return r, nil
}

func Hello(c echo.Context) error {
	return httpx.RestAbort(c, "hello world", nil)
}
