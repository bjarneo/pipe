package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// File permission constants
const (
	// logFilePermissions restricts log file access to owner only (read/write)
	// to prevent other users from reading potentially sensitive deployment logs
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

// Logger handles logging to both console and file
type Logger struct {
	writer  io.Writer
	file    *os.File // Keep reference for Close()
	verbose bool
	quiet   bool // suppress console output (for JSON mode)
}

// Ensure Logger implements Interface
var _ Interface = (*Logger)(nil)

// New creates a new logger instance that writes to a file
func New(filename string, verbose bool) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFilePermissions)
	if err != nil {
		return nil, err
	}
	return &Logger{writer: file, file: file, verbose: verbose}, nil
}

// NewWithWriter creates a new logger instance that writes to any io.Writer
// Useful for testing or custom output destinations
func NewWithWriter(w io.Writer, verbose bool) *Logger {
	return &Logger{writer: w, file: nil, verbose: verbose}
}

// SetQuiet enables or disables quiet mode (suppresses console output)
func (l *Logger) SetQuiet(quiet bool) {
	if l != nil {
		l.quiet = quiet
	}
}

// Step logs a deployment step (always shown unless quiet)
func (l *Logger) Step(message string) error {
	if l == nil || l.writer == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] STEP: %s\n", timestamp, message)
	if !l.quiet {
		fmt.Printf("> %s\n", message)
	}
	_, err := l.writer.Write([]byte(logMessage))
	return err
}

// StepProgress logs a numbered deployment step with status
func (l *Logger) StepProgress(stepNum, total int, action, status string) error {
	if l == nil || l.writer == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] STEP [%d/%d]: %s... %s\n", timestamp, stepNum, total, action, status)
	if !l.quiet {
		fmt.Printf("[%d/%d] %s... %s\n", stepNum, total, action, status)
	}
	_, err := l.writer.Write([]byte(logMessage))
	return err
}

// Info logs an informational message (only in verbose mode on console)
func (l *Logger) Info(message string) error {
	if l == nil || l.writer == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] INFO: %s\n", timestamp, message)
	if l.verbose && !l.quiet {
		fmt.Println(message)
	}
	_, err := l.writer.Write([]byte(logMessage))
	return err
}

// Debug logs a debug message (only in verbose mode)
func (l *Logger) Debug(message string) error {
	if l == nil || l.writer == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] DEBUG: %s\n", timestamp, message)
	if l.verbose && !l.quiet {
		fmt.Printf("  %s\n", message)
	}
	_, err := l.writer.Write([]byte(logMessage))
	return err
}

// Error logs an error message
func (l *Logger) Error(message string, err error) error {
	if l == nil || l.writer == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n%s\n", timestamp, message, errStr)
	fmt.Printf("ERROR: %s\n", message)
	if err != nil {
		fmt.Printf("  Error: %s\n", err)
	}
	_, writeErr := l.writer.Write([]byte(logMessage))
	return writeErr
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(err error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] FATAL: %s\n", timestamp, err.Error())
	fmt.Printf("FATAL: %s\n", err)
	if l != nil && l.writer != nil {
		_, _ = l.writer.Write([]byte(logMessage))
		// Sync if we have a file
		if l.file != nil {
			_ = l.file.Sync()
		}
		_ = l.Close()
	}
	os.Exit(1)
}

// IsVerbose returns whether verbose mode is enabled
func (l *Logger) IsVerbose() bool {
	return l != nil && l.verbose
}

// Close closes the log file if one was opened
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
} 