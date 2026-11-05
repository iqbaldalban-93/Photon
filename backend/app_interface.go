package backend

import "log/slog"

type AppInterface interface {
	EmitEvent(event string, data any)

	GetLogger() *slog.Logger
}
