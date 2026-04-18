package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

type Logger struct {
	logs    chan string
	enabled bool
	writer  io.Writer
}

func NewLogger(bufferSize int, enabled bool) *Logger {
	return &Logger{
		logs:    make(chan string, bufferSize),
		enabled: enabled,
		writer:  os.Stdout,
	}
}

func (l *Logger) Start() {
	if !l.enabled {
		return
	}
	go func() {
		for msg := range l.logs {
			fmt.Fprintf(l.writer, "[%s] %s\n", time.Now().Format(time.RFC3339), msg)
		}
	}()
}

func (l *Logger) Log(msg string) {
	if l.enabled {
		// Non-blocking send
		select {
		case l.logs <- msg:
		default:
			// Buffer full, skip log to preserve performance
		}
	}
}

func (l *Logger) Stop() {
	close(l.logs)
}
