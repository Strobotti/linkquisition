package linkquisition

import (
	"context"
	"log/slog"
)

type Application interface {
	Run(ctx context.Context) error
	GetLogger() *slog.Logger
}
