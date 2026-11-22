package handlers

import (
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/ratelimit"
	"github.com/kaekkr/kladovki/internal/service"
)

type Handler struct {
	svc    *service.Service
	tokens *service.TokenService
	cfg    config.Config
	mw     *middleware.Auth
	limits limiters
}

// limiters are in-memory, per process (see internal/ratelimit).
type limiters struct {
	auth, lead, lock *ratelimit.Limiter
	loginFailures    *ratelimit.Limiter
}

func newLimiters(c config.RateLimits) limiters {
	mk := func(l config.RateLimit) *ratelimit.Limiter { return ratelimit.New(l.Requests, l.Window) }
	return limiters{auth: mk(c.Auth), lead: mk(c.Lead), lock: mk(c.Lock), loginFailures: mk(c.LoginFailures)}
}

func New(svc *service.Service, tokens *service.TokenService, cfg config.Config, mw *middleware.Auth) *Handler {
	return &Handler{svc: svc, tokens: tokens, cfg: cfg, mw: mw, limits: newLimiters(cfg.RateLimits)}
}
