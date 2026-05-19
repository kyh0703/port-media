package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type RequestLogger struct {
	log *zap.Logger
}

func NewRequestLogger(log *zap.Logger) *RequestLogger {
	return &RequestLogger{log: log}
}

func (m *RequestLogger) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		m.log.Info("http_request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
		)
		return err
	}
}
