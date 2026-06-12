// Package logger provides a shared, production-grade logger that writes
// simultaneously to stdout and a persistent log file (logs/app.log).
// All packages import this instead of using log.Printf directly so that
// every log line ends up in the file without any extra wiring.
package logger

import (
	"io"
	"log"
	"os"
)

// App is the shared logger used across the whole application.
var App *log.Logger

// Init opens (or creates) logs/app.log and wires App to fan-out to both
// stdout and the file. Call this once from main before starting the server.
func Init() (closeFunc func(), err error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	mw := io.MultiWriter(os.Stdout, f)
	App = log.New(mw, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)

	return func() { _ = f.Close() }, nil
}
