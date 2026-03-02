package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var Log *slog.Logger

func init() {
	// Initialize with a safe default (stderr, info level)
	// This ensures Log is never nil even if Init fails or isn't called
	Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(Log)
}

type Options struct {
	Level    string // "debug", "info", "warn", "error"
	FilePath string // path to log file
	Console  bool   // output to console as well
	File     bool   // output to file
}

func Init(opts Options) error {
	var level slog.Level
	switch opts.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var writers []io.Writer

	if opts.Console {
		writers = append(writers, os.Stderr)
	}

	if opts.File && opts.FilePath != "" {
		dir := filepath.Dir(opts.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// If we can't create the directory, we'll return the error
			// but we WON'T set Log to nil. We'll just stick with stderr
			// or update the level on the default logger.
			// Let's try to update the level at least.
			if len(writers) == 0 {
				writers = append(writers, os.Stderr)
			}
			updateDefaultLogger(io.MultiWriter(writers...), level)
			return err
		}

		f, err := os.OpenFile(opts.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			if len(writers) == 0 {
				writers = append(writers, os.Stderr)
			}
			updateDefaultLogger(io.MultiWriter(writers...), level)
			return err
		}
		writers = append(writers, f)
	}

	if len(writers) == 0 {
		// Default to stderr if nothing enabled
		writers = append(writers, os.Stderr)
	}

	w := io.MultiWriter(writers...)
	updateDefaultLogger(w, level)
	return nil
}

func updateDefaultLogger(w io.Writer, level slog.Level) {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// Helper functions

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

func Debugf(format string, args ...any) {
	Log.Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	Log.Info(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	Log.Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	Log.Error(fmt.Sprintf(format, args...))
}
