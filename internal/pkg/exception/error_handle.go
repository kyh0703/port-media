package exception

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/zap"
)

func NewErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.Status).JSON(response.Error(appErr.Status, appErr.Message, appErr.Code))
		}

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return c.Status(fiberErr.Code).JSON(response.Error(fiberErr.Code, fiberErr.Message, CodeBadRequest))
		}

		log.Error("unhandled request error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(
			response.Error(fiber.StatusInternalServerError, "internal server error", CodeInternalError),
		)
	}
}
