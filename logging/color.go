package logging

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	terminal "golang.org/x/term"
)

var colorOrder = []string{
	"36", "33", "32", "35", "31", "34",
	"36;1", "33;1", "32;1", "35;1", "31;1", "34;1",
}

// ColorLoggerFactory creates consistently aligned, colorized container loggers.
type ColorLoggerFactory struct {
	mu        sync.RWMutex
	maxLength int
	nextColor int
	tty       bool
}

// ColorLogger prefixes output with a container or service name.
type ColorLogger struct {
	name        string
	colorPrefix string
	factory     *ColorLoggerFactory
}

// NewColorLoggerFactory creates a logger factory for the current terminal.
func NewColorLoggerFactory() *ColorLoggerFactory {
	return &ColorLoggerFactory{tty: terminal.IsTerminal(int(os.Stdout.Fd()))}
}

func (factory *ColorLoggerFactory) CreateContainerLogger(name string) Logger {
	return factory.create(name)
}

func (*ColorLoggerFactory) CreateBuildLogger(string) Logger { return &RawLogger{} }
func (*ColorLoggerFactory) CreatePullLogger(string) Logger  { return &NullLogger{} }

func (factory *ColorLoggerFactory) create(name string) Logger {
	factory.mu.Lock()
	if factory.maxLength < len(name) {
		factory.maxLength = len(name)
	}
	color := colorOrder[factory.nextColor]
	factory.nextColor = (factory.nextColor + 1) % len(colorOrder)
	factory.mu.Unlock()

	return &ColorLogger{
		name:        name,
		factory:     factory,
		colorPrefix: fmt.Sprintf("\x1b[%sm%%s |\x1b[0m", color),
	}
}

func (logger *ColorLogger) Out(data []byte) {
	if len(data) > 0 {
		fmt.Print(logger.format(string(data)))
	}
}

func (logger *ColorLogger) Err(data []byte) {
	if len(data) > 0 {
		fmt.Fprint(os.Stderr, logger.format(string(data)))
	}
}

func (*ColorLogger) OutWriter() io.Writer { return os.Stdout }
func (*ColorLogger) ErrWriter() io.Writer { return os.Stderr }

func (logger *ColorLogger) format(message string) string {
	logger.factory.mu.RLock()
	pad := logger.factory.maxLength
	tty := logger.factory.tty
	logger.factory.mu.RUnlock()

	name := fmt.Sprintf("%-"+strconv.Itoa(pad)+"s", logger.name)
	if tty {
		return fmt.Sprintf(logger.colorPrefix+" %s", name, message)
	}
	return fmt.Sprintf("%s | %s", name, message)
}
