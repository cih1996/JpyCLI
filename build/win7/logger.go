package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger 兼容 Go 1.19 的日志封装
type Logger struct {
	inner *log.Logger
	level Level
}

var Log *Logger

func init() {
	Log = &Logger{
		inner: log.New(os.Stderr, "", log.LstdFlags),
		level: LevelInfo,
	}
}

type Options struct {
	Level    string
	FilePath string
	Console  bool
	File     bool
}

func Init(opts Options) error {
	level := parseLevel(opts.Level)
	var writers []io.Writer

	if opts.Console {
		writers = append(writers, os.Stderr)
	}

	if opts.File && opts.FilePath != "" {
		dir := filepath.Dir(opts.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			if len(writers) == 0 {
				writers = append(writers, os.Stderr)
			}
			Log = &Logger{inner: log.New(io.MultiWriter(writers...), "", log.LstdFlags), level: level}
			return err
		}
		f, err := os.OpenFile(opts.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			if len(writers) == 0 {
				writers = append(writers, os.Stderr)
			}
			Log = &Logger{inner: log.New(io.MultiWriter(writers...), "", log.LstdFlags), level: level}
			return err
		}
		writers = append(writers, f)
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stderr)
	}

	Log = &Logger{inner: log.New(io.MultiWriter(writers...), "", log.LstdFlags), level: level}
	return nil
}

func parseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l *Logger) log(lvl Level, prefix string, msg string, args ...any) {
	if lvl < l.level {
		return
	}
	if len(args) > 0 {
		l.inner.Printf("%s %s %v", prefix, msg, args)
	} else {
		l.inner.Printf("%s %s", prefix, msg)
	}
}

func (l *Logger) Debug(msg string, args ...any) { l.log(LevelDebug, "[DEBUG]", msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.log(LevelInfo, "[INFO]", msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.log(LevelWarn, "[WARN]", msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.log(LevelError, "[ERROR]", msg, args...) }

func Debug(msg string, args ...any) { Log.Debug(msg, args...) }
func Info(msg string, args ...any)  { Log.Info(msg, args...) }
func Warn(msg string, args ...any)  { Log.Warn(msg, args...) }
func Error(msg string, args ...any) { Log.Error(msg, args...) }

func Debugf(format string, args ...any) { Log.Debug(fmt.Sprintf(format, args...)) }
func Infof(format string, args ...any)  { Log.Info(fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...any)  { Log.Warn(fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...any) { Log.Error(fmt.Sprintf(format, args...)) }
