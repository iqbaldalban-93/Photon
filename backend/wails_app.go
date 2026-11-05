package backend

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type WailsApp struct {
	app *application.App

	OnEvent func(event string, data any)
}

func NewWailsApp(app *application.App) *WailsApp {
	return &WailsApp{app: app}
}

func (w *WailsApp) EmitEvent(event string, data any) {
	if w.OnEvent != nil {
		w.OnEvent(event, data)
	}
	w.app.Event.Emit(event, data)
}

func (w *WailsApp) GetLogger() *slog.Logger {
	return w.app.Logger
}
