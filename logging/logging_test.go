package logging

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

type captureLogger struct {
	out bytes.Buffer
	err bytes.Buffer
}

func (logger *captureLogger) Out(data []byte)      { _, _ = logger.out.Write(data) }
func (logger *captureLogger) Err(data []byte)      { _, _ = logger.err.Write(data) }
func (logger *captureLogger) OutWriter() io.Writer { return &logger.out }
func (logger *captureLogger) ErrWriter() io.Writer { return &logger.err }

func TestWrapperRoutesStreams(t *testing.T) {
	logger := &captureLogger{}
	out := &Wrapper{Logger: logger}
	err := &Wrapper{Err: true, Logger: logger}
	if _, writeErr := out.Write([]byte("out")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, writeErr := err.Write([]byte("err")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if logger.out.String() != "out" || logger.err.String() != "err" {
		t.Fatalf("unexpected output: out=%q err=%q", logger.out.String(), logger.err.String())
	}
}

func TestColorFactoryConcurrentCreation(t *testing.T) {
	factory := NewColorLoggerFactory()
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_ = factory.CreateContainerLogger("service")
		}()
	}
	wait.Wait()
}
