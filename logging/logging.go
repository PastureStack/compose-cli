// Package logging provides the output adapters used by Compose operations.
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// SafeLogValue converts an external value to a single log record. Terminal
// output adapters intentionally do not use this function because preserving
// their line structure is part of their public behavior.
func SafeLogValue(value interface{}) string {
	text := fmt.Sprint(value)
	text = strings.ReplaceAll(text, "\r", "")
	return strings.ReplaceAll(text, "\n", " ")
}

// Factory creates loggers for containers, builds, and image pulls.
type Factory interface {
	CreateContainerLogger(name string) Logger
	CreateBuildLogger(name string) Logger
	CreatePullLogger(name string) Logger
}

// Logger writes standard and error output for an operation.
type Logger interface {
	Out([]byte)
	Err([]byte)
	OutWriter() io.Writer
	ErrWriter() io.Writer
}

// Wrapper adapts a Logger to io.Writer.
type Wrapper struct {
	Err    bool
	Logger Logger
}

// Write forwards bytes to the selected logger stream.
func (wrapper *Wrapper) Write(data []byte) (int, error) {
	if wrapper.Err {
		wrapper.Logger.Err(data)
	} else {
		wrapper.Logger.Out(data)
	}
	return len(data), nil
}

// NullLogger discards all output and also implements Factory.
type NullLogger struct{}

func (*NullLogger) Out([]byte)                          {}
func (*NullLogger) Err([]byte)                          {}
func (*NullLogger) OutWriter() io.Writer                { return nil }
func (*NullLogger) ErrWriter() io.Writer                { return nil }
func (*NullLogger) CreateContainerLogger(string) Logger { return &NullLogger{} }
func (*NullLogger) CreateBuildLogger(string) Logger     { return &NullLogger{} }
func (*NullLogger) CreatePullLogger(string) Logger      { return &NullLogger{} }

// RawLogger writes bytes without prefixes and also implements Factory.
type RawLogger struct{}

func (*RawLogger) Out(data []byte)                     { fmt.Print(string(data)) }
func (*RawLogger) Err(data []byte)                     { fmt.Fprint(os.Stderr, string(data)) }
func (*RawLogger) OutWriter() io.Writer                { return os.Stdout }
func (*RawLogger) ErrWriter() io.Writer                { return os.Stderr }
func (*RawLogger) CreateContainerLogger(string) Logger { return &RawLogger{} }
func (*RawLogger) CreateBuildLogger(string) Logger     { return &RawLogger{} }
func (*RawLogger) CreatePullLogger(string) Logger      { return &RawLogger{} }
