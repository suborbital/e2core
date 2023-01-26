package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func UUIDRequestID() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return uuid.New().String()
		},
		RequestIDHandler: func(c echo.Context, s string) {
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, "requestID", s)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Set("requestID", s)
		},
	})
}
