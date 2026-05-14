package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/toanehihi/object-storage/internal/token"
)

const _contextKeyUserID = "userID"

func JWTAuth(tm *token.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			tokenString, found := strings.CutPrefix(header, "Bearer ")
			if !found {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			claims, err := tm.Validate(tokenString)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(_contextKeyUserID, claims.UserID)
			return next(c)
		}
	}
}

func UserIDFromContext(c echo.Context) uuid.UUID {
	id, ok := c.Get(_contextKeyUserID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
