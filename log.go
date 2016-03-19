package main

import (
	"fmt"
	"io"
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

// ConsoleLogger implements
type ConsoleLogger struct {
	W io.Writer
}

// SetOutput sets the output to w
func (nl *ConsoleLogger) SetOutput(w io.Writer) {
	nl.W = w
}

func (nl ConsoleLogger) output() io.Writer {
	if nl.W == nil {
		return os.Stderr
	}
	return nl.W
}

// Debug in the null logger does nothing
func (nl ConsoleLogger) Debug(format string, a ...interface{}) {
	fmt.Fprintf(nl.output(), format, a...)
}

// Info in the null logger does nothing
func (nl ConsoleLogger) Info(format string, a ...interface{}) {
	fmt.Fprintf(nl.output(), format, a...)
}

// Warning in the null logger does nothing
func (nl ConsoleLogger) Warning(format string, a ...interface{}) {
	fmt.Fprintf(nl.output(), format, a...)
}

// Error in the null logger does nothing
func (nl ConsoleLogger) Error(format string, a ...interface{}) {
	fmt.Fprintf(nl.output(), format, a...)
}

// Fatalf in the null logger does nothing
func (nl ConsoleLogger) Fatalf(format string, a ...interface{}) {
	fmt.Fprintf(nl.output(), format, a...)
	os.Exit(1)
}

// Level in the null logger does nothing
func (nl ConsoleLogger) Level(level LogLevel) {}
