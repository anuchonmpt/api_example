package handler

import (
	"context"
	"net/http"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/middleware"
	"github.com/example/api-example/internal/service"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/gin-gonic/gin"
)

type authService interface {
	Register(context.Context, service.RegisterInput) (*service.AuthResult, error)
	Login(context.Context, service.LoginInput) (*service.AuthResult, error)
	Refresh(context.Context, string) (*service.AuthResult, error)
	Logout(context.Context, string) error
	Me(context.Context, int64) (*domain.User, error)
}

type Auth struct{ service authService }

func NewAuth(service authService) *Auth { return &Auth{service: service} }

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Auth) Register(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		apperrors.WriteGin(c, apperrors.WithMessage(apperrors.ErrAuthInvalidBody, "email and password are required"))
		return
	}
	result, err := h.service.Register(c.Request.Context(), service.RegisterInput{Email: request.Email, Password: request.Password})
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Auth) Login(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		apperrors.WriteGin(c, apperrors.ErrAuthInvalidBody)
		return
	}
	result, err := h.service.Login(c.Request.Context(), service.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Auth) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		apperrors.WriteGin(c, apperrors.ErrAuthInvalidBody)
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Auth) Logout(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		apperrors.WriteGin(c, apperrors.ErrAuthInvalidBody)
		return
	}
	if err := h.service.Logout(c.Request.Context(), request.RefreshToken); err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Auth) Me(c *gin.Context) {
	actor, ok := middleware.GetActor(c)
	if !ok {
		apperrors.WriteGin(c, apperrors.ErrAuthUnauthorized)
		return
	}
	user, err := h.service.Me(c.Request.Context(), actor.UserID)
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}
