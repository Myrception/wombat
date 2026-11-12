package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// writeCrashLog appends the panic value and stack of the current goroutine to
// <appData>/crash.log and returns the log's path.
func writeCrashLog(appData string, r interface{}) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "=== %s %s (%s)\n%+v\n\n", time.Now().Format(time.RFC3339), appName, semver, r)
	b.Write(debug.Stack())
	b.WriteString("\n")

	fmt.Fprint(os.Stderr, b.String())
	if appData == "" {
		return ""
	}
	p := filepath.Join(appData, "crash.log")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return ""
	}
	defer f.Close()
	f.Write(b.Bytes())
	return p
}

// recoverPanic must be deferred directly. It turns a panic in a background
// goroutine into a crash log entry and an error dialog instead of killing the
// whole application.
func (a *api) recoverPanic(where string) {
	r := recover()
	if r == nil {
		return
	}
	msg := fmt.Sprintf("unexpected error in %s: %v", where, r)
	if p := writeCrashLog(a.appData, r); p != "" {
		msg += fmt.Sprintf(" (details in %s)", p)
	}
	if a.rt != nil {
		a.rt.LogError(msg)
		a.emitError("Internal error", msg)
	}
}

// goSafe runs fn on a new goroutine, recovering from panics.
func (a *api) goSafe(where string, fn func()) {
	go func() {
		defer a.recoverPanic(where)
		fn()
	}()
}
