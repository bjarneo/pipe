package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const (
	logFilePermissions = 0600
)

// Interface defines the logging interface for dependency injection and testing
type Interface interface {
	Info(message string) error
	Step(message string) error
	StepProgress(stepNum, total int, action, status string) error
	Debug(message string) error
	Error(message string, err error) error
	SetQuiet(quiet bool)
	IsVerbose() bool
	Close() error
}

// Logger handles logging to both console and file using slog
type Logger struct {
	slog    *slog.Logger
	file    *os.File
	verbose bool
	quiet   bool
}

var _ Interface = (*Logger)(nil)

func New(filename string, verbose bool) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFilePermissions)
	if err != nil {
		return nil, err
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})
	return &Logger{
		slog:    slog.New(handler),
		file:    file,
		verbose: verbose,
	}, nil
}

// NewWithWriter creates a new logger that writes to any io.Writer (useful for testing)
func NewWithWriter(w io.Writer, verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return &Logger{
		slog:    slog.New(handler),
		file:    nil,
		verbose: verbose,
	}
}

func (l *Logger) SetQuiet(quiet bool) {
	if l != nil {
		l.quiet = quiet
	}
}

func (l *Logger) Step(message string) error {
	if l == nil || l.slog == nil {
		return nil
	}
	l.slog.LogAttrs(context.Background(), slog.LevelInfo, message,
		slog.String("type", "step"))
	if !l.quiet {
		fmt.Printf("> %s\n", message)
	}
	return nil
}

func (l *Logger) StepProgress(stepNum, total int, action, status string) error {
	if l == nil || l.slog == nil {
		return nil
	}
	l.slog.LogAttrs(context.Background(), slog.LevelInfo, action,
		slog.String("type", "progress"),
		slog.Int("step", stepNum),
		slog.Int("total", total),
		slog.String("status", status))
	if !l.quiet {
		fmt.Printf("[%d/%d] %s... %s\n", stepNum, total, action, status)
	}
	return nil
}

func (l *Logger) Info(message string) error {
	if l == nil || l.slog == nil {
		return nil
	}
	l.slog.Info(message)
	if l.verbose && !l.quiet {
		fmt.Println(message)
	}
	return nil
}

func (l *Logger) Debug(message string) error {
	if l == nil || l.slog == nil {
		return nil
	}
	l.slog.Debug(message)
	if l.verbose && !l.quiet {
		fmt.Printf("  %s\n", message)
	}
	return nil
}

func (l *Logger) Error(message string, err error) error {
	if l == nil || l.slog == nil {
		return nil
	}
	if err != nil {
		l.slog.Error(message, slog.Any("error", err))
	} else {
		l.slog.Error(message)
	}
	fmt.Printf("ERROR: %s\n", message)
	if err != nil {
		fmt.Printf("  Error: %s\n", err)
	}
	return nil
}

func (l *Logger) Fatal(err error) {
	if l != nil && l.slog != nil {
		l.slog.Error("fatal error", slog.Any("error", err))
		if l.file != nil {
			_ = l.file.Sync()
		}
		_ = l.Close()
	}
	fmt.Printf("FATAL: %s\n", err)
	os.Exit(1)
}

func (l *Logger) IsVerbose() bool {
	return l != nil && l.verbose
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
