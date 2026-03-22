package logger

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
)

type ctxKey struct{}

const callerPadWidth = 50

// New creates a zerolog.Logger configured from LogConfig.
//
// Console output format:
//
//	2026-03-21 00:29:10 INF internal/logger/logger.go:42            --> Logger initialized LOG_LEVEL=debug
func New(cfg config.LogConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Override zerolog global settings
	zerolog.TimeFieldFormat = time.DateTime
	zerolog.CallerMarshalFunc = shortCaller

	if cfg.Format == "json" {
		return zerolog.New(os.Stdout).Level(level).With().Timestamp().Caller().Logger()
	}

	w := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		NoColor:    false,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%-3s", i))
		},
		FormatCaller: func(i interface{}) string {
			caller := fmt.Sprintf("%s", i)
			if pad := callerPadWidth - len(caller); pad > 0 {
				caller += strings.Repeat(" ", pad)
			}
			return caller + " -->"
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf(" %s", i)
		},
	}

	l := zerolog.New(w).Level(level).With().Timestamp().Caller().Logger()
	l.Info().Str("LOG_LEVEL", cfg.Level).Msg("Logger initialized")

	return l
}

// shortCaller trims the caller path to be relative to the module root.
func shortCaller(pc uintptr, file string, line int) string {
	// Trim everything before "tax-module/" or "internal/"
	if idx := strings.Index(file, "internal/"); idx != -1 {
		file = file[idx:]
	} else if idx := strings.Index(file, "cmd/"); idx != -1 {
		file = file[idx:]
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// FromContext extracts the logger stored in ctx.
// Falls back to a disabled logger if none is found.
func FromContext(ctx context.Context) *zerolog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zerolog.Logger); ok {
		return l
	}
	nop := zerolog.Nop()
	return &nop
}

// WithContext returns a new context with the given logger attached.
func WithContext(ctx context.Context, l *zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}
