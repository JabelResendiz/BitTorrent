package internal

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Colors ANSI
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// Logger represents a single module string
type Logger struct {
	module string
}

// Constructor: para que cada módulo tenga su propio tag
func NewLogger(module string) *Logger {
	log.SetFlags(0) 
	return &Logger{module: module}
}

// Format the output
func (l *Logger) log(color string, level string, msg string, args ...any) {
	timestamp := time.Now().Format("15:04:05.000")
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(os.Stdout, "%s[%s]%s %s%-5s%s [%s] %s\n",
		colorCyan, timestamp, colorReset,
		color, level, colorReset,
		l.module,
		formatted,
	)
}

// Methods
func (l *Logger) Info(msg string, args ...any)  { l.log(colorGreen, "INFO", msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.log(colorYellow, "WARN", msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.log(colorRed, "ERROR", msg, args...) }
func (l *Logger) Debug(msg string, args ...any) { l.log(colorBlue, "DEBUG", msg, args...) }
