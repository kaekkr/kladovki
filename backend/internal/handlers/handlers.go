package handlers

import (
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/service"
)

type Handler struct {
	svc    *service.Service
	tokens *service.TokenService
	cfg    config.Config
	mw     *middleware.Auth
}

func New(svc *service.Service, tokens *service.TokenService, cfg config.Config, mw *middleware.Auth) *Handler {
	return &Handler{svc: svc, tokens: tokens, cfg: cfg, mw: mw}
}
