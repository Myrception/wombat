package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// wailsRuntime is the subset of the Wails runtime used by the api. It exists
// so the api can be exercised in tests without a running webview.
type wailsRuntime interface {
	Emit(event string, data ...interface{})
	LogDebug(msg string)
	LogInfo(msg string)
	LogWarning(msg string)
	LogError(msg string)
}

// ctxRuntime forwards to the real Wails runtime bound to ctx.
type ctxRuntime struct{ ctx context.Context }

func (r ctxRuntime) Emit(event string, data ...interface{}) {
	runtime.EventsEmit(r.ctx, event, data...)
}
func (r ctxRuntime) LogDebug(msg string)   { runtime.LogDebug(r.ctx, msg) }
func (r ctxRuntime) LogInfo(msg string)    { runtime.LogInfo(r.ctx, msg) }
func (r ctxRuntime) LogWarning(msg string) { runtime.LogWarning(r.ctx, msg) }
func (r ctxRuntime) LogError(msg string)   { runtime.LogError(r.ctx, msg) }
