package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/toanehihi/object-storage/internal/middleware"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/internal/service"
	"github.com/toanehihi/object-storage/internal/token"
	"github.com/toanehihi/object-storage/internal/utils/response"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group, authMW echo.MiddlewareFunc) {
	g.POST("/register", h.register)
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
	g.GET("/me", h.me, authMW)
}

func (h *AuthHandler) register(c echo.Context) error {
	var req service.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	resp, err := h.svc.Register(c.Request().Context(), req)
	if errors.Is(err, service.ErrEmailTaken) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "registration failed")
	}

	return c.JSON(http.StatusCreated, response.OK(resp, "user registered successfully"))
}

func (h *AuthHandler) login(c echo.Context) error {
	var req service.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if errors.Is(err, service.ErrInvalidPassword) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	return c.JSON(http.StatusOK, response.OK(resp, "login successful"))
}

func (h *AuthHandler) refresh(c echo.Context) error {
	var req service.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.RefreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "refresh token is required")
	}

	tokens, err := h.svc.Refresh(c.Request().Context(), req)
	if errors.Is(err, token.ErrInvalidToken) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token refresh failed")
	}

	return c.JSON(http.StatusOK, response.OK(tokens, "token refreshed"))
}

func (h *AuthHandler) me(c echo.Context) error {
	userID := middleware.UserIDFromContext(c)

	user, err := h.svc.GetProfile(c.Request().Context(), userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get profile")
	}

	return c.JSON(http.StatusOK, response.OK(*user, ""))
}
