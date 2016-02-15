package main

import (
	"fmt"
	"os"
)

// LogLevel identifies logging levels
type LogLevel int

// Definitions of log lgevels
const (
	Debug LogLevel = iota
	Info
	Warning
	Error
	Fatal
)

// Logger is the interface used for logging
type Logger interface {
	// Info loggs a message of info level
	Info(format string, a ...interface{})
	Warning(format string, a ...interface{})
	Error(format string, a ...interface{})
	Fatalf(format string, a ...interface{})
	Debug(format string, a ...interface{})

	Level(level LogLevel)
}

// NullLogger implements
type NullLogger struct{}

// Info in the null logger does nothing
func (nl NullLogger) Info(format string, a ...interface{}) {}

// Warning in the null logger does nothing
func (nl NullLogger) Warning(format string, a ...interface{}) {}

// Error in the null logger does nothing
func (nl NullLogger) Error(format string, a ...interface{}) {}

// Fatalf in the null logger does nothing
func (nl NullLogger) Fatalf(format string, a ...interface{}) {
	os.Exit(1)
}

// Debug in the null logger does nothing
func (nl NullLogger) Debug(format string, a ...interface{}) {}

// Level in the null logger does nothing
func (nl NullLogger) Level(level LogLevel) {}

// ConsoleLogger implements
type ConsoleLogger struct{}

// Debug in the null logger does nothing
func (nl ConsoleLogger) Debug(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

// Info in the null logger does nothing
func (nl ConsoleLogger) Info(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

// Warning in the null logger does nothing
func (nl ConsoleLogger) Warning(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

// Error in the null logger does nothing
func (nl ConsoleLogger) Error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

// Fatalf in the null logger does nothing
func (nl ConsoleLogger) Fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

// Level in the null logger does nothing
func (nl ConsoleLogger) Level(level LogLevel) {}
