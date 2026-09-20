package ports

import (
	"context"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

// Pipeline is the engine's store surface. It does not expose the pool.
type Pipeline interface {
	Logger
	Settings(ctx context.Context) (kernel.Settings, error)
	PendingMessages(ctx context.Context, limit int) ([]kernel.Message, error)
	HashExists(ctx context.Context, hash string, exceptID int64) (bool, error)
	ClaimProcessing(ctx context.Context, id int64) (bool, error)
	FinishMessage(ctx context.Context, id int64, llmStatus, errMsg string, result any, importance float64) error
}
