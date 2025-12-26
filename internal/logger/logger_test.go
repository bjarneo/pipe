package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	if log == nil {
		t.Fatal("NewWithWriter() returned nil")
	}
	if log.verbose {
		t.Error("NewWithWriter() verbose should be false")
	}
	if log.quiet {
		t.Error("NewWithWriter() quiet should be false by default")
	}
}

func TestNewWithWriter_Verbose(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, true)

	if !log.verbose {
		t.Error("NewWithWriter() verbose should be true")
	}
}

func TestLogger_SetQuiet(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	log.SetQuiet(true)
	if !log.quiet {
		t.Error("SetQuiet(true) should set quiet to true")
	}

	log.SetQuiet(false)
	if log.quiet {
		t.Error("SetQuiet(false) should set quiet to false")
	}
}

func TestLogger_SetQuiet_NilLogger(t *testing.T) {
	var log *Logger
	// Should not panic
	log.SetQuiet(true)
}

func TestLogger_Step(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	err := log.Step("Test step")
	if err != nil {
		t.Errorf("Step() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test step") {
		t.Errorf("Step() output should contain message, got %q", output)
	}
	if !strings.Contains(output, `"type":"step"`) {
		t.Errorf("Step() output should contain type:step, got %q", output)
	}
}

func TestLogger_Step_Quiet(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)
	log.SetQuiet(true)

	err := log.Step("Test step")
	if err != nil {
		t.Errorf("Step() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test step") {
		t.Errorf("Step() should still log to writer in quiet mode, got %q", output)
	}
}

func TestLogger_Step_NilLogger(t *testing.T) {
	var log *Logger
	err := log.Step("Test")
	if err != nil {
		t.Errorf("Step() on nil logger should return nil, got %v", err)
	}
}

func TestLogger_StepProgress(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	err := log.StepProgress(1, 4, "Building", "done")
	if err != nil {
		t.Errorf("StepProgress() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Building") {
		t.Errorf("StepProgress() output should contain action, got %q", output)
	}
	if !strings.Contains(output, `"step":1`) {
		t.Errorf("StepProgress() output should contain step number, got %q", output)
	}
	if !strings.Contains(output, `"total":4`) {
		t.Errorf("StepProgress() output should contain total, got %q", output)
	}
	if !strings.Contains(output, `"status":"done"`) {
		t.Errorf("StepProgress() output should contain status, got %q", output)
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	err := log.Info("Test info")
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("Info() output should contain level INFO, got %q", output)
	}
	if !strings.Contains(output, "Test info") {
		t.Errorf("Info() output should contain message, got %q", output)
	}
}

func TestLogger_Info_NilLogger(t *testing.T) {
	var log *Logger
	err := log.Info("Test")
	if err != nil {
		t.Errorf("Info() on nil logger should return nil, got %v", err)
	}
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, true) // verbose to enable debug

	err := log.Debug("Test debug")
	if err != nil {
		t.Errorf("Debug() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"level":"DEBUG"`) {
		t.Errorf("Debug() output should contain level DEBUG, got %q", output)
	}
	if !strings.Contains(output, "Test debug") {
		t.Errorf("Debug() output should contain message, got %q", output)
	}
}

func TestLogger_Debug_NilLogger(t *testing.T) {
	var log *Logger
	err := log.Debug("Test")
	if err != nil {
		t.Errorf("Debug() on nil logger should return nil, got %v", err)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	err := log.Error("Test error", nil)
	if err != nil {
		t.Errorf("Error() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"level":"ERROR"`) {
		t.Errorf("Error() output should contain level ERROR, got %q", output)
	}
	if !strings.Contains(output, "Test error") {
		t.Errorf("Error() output should contain message, got %q", output)
	}
}

func TestLogger_Error_WithError(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	testErr := &testError{msg: "underlying error"}
	err := log.Error("Test error", testErr)
	if err != nil {
		t.Errorf("Error() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "underlying error") {
		t.Errorf("Error() output should contain underlying error, got %q", output)
	}
}

func TestLogger_Error_NilLogger(t *testing.T) {
	var log *Logger
	err := log.Error("Test", nil)
	if err != nil {
		t.Errorf("Error() on nil logger should return nil, got %v", err)
	}
}

func TestLogger_IsVerbose(t *testing.T) {
	var buf bytes.Buffer

	log := NewWithWriter(&buf, false)
	if log.IsVerbose() {
		t.Error("IsVerbose() should return false for non-verbose logger")
	}

	log = NewWithWriter(&buf, true)
	if !log.IsVerbose() {
		t.Error("IsVerbose() should return true for verbose logger")
	}
}

func TestLogger_IsVerbose_NilLogger(t *testing.T) {
	var log *Logger
	if log.IsVerbose() {
		t.Error("IsVerbose() on nil logger should return false")
	}
}

func TestLogger_Close_NilFile(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	err := log.Close()
	if err != nil {
		t.Errorf("Close() with nil file should return nil, got %v", err)
	}
}

func TestLogger_Close_NilLogger(t *testing.T) {
	var log *Logger
	err := log.Close()
	if err != nil {
		t.Errorf("Close() on nil logger should return nil, got %v", err)
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf, false)

	_ = log.Info("Test")

	output := buf.String()
	// Check for JSON format with time field
	if !strings.Contains(output, `"time"`) {
		t.Errorf("Output should be JSON with time field, got %q", output)
	}
	if !strings.Contains(output, `"msg"`) {
		t.Errorf("Output should be JSON with msg field, got %q", output)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestLoggerInterface(t *testing.T) {
	var _ Interface = (*Logger)(nil)
}
