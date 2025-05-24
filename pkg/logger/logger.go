// Package logger provides colored logging utilities for console output.
// It supports different log levels (info, warn, error, debug, success) with color coding
// and provides fatal logging that exits the application.
package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	infoColor    = color.New(color.FgGreen)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	debugColor   = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen, color.Bold)
)

// Info prints an informational message in green color.
func Info(format string, args ...interface{}) {
	infoColor.Printf(format, args...)
}

// Infof prints an informational message in green color (alias for Info).
func Infof(format string, args ...interface{}) {
	infoColor.Printf(format, args...)
}

// Infoln prints an informational message in green color with a newline.
func Infoln(format string, args ...interface{}) {
	infoColor.Printf(format+"\n", args...)
}

// Warn prints a warning message in yellow color.
func Warn(format string, args ...interface{}) {
	warnColor.Printf(format, args...)
}

// Warnf prints a warning message in yellow color (alias for Warn).
func Warnf(format string, args ...interface{}) {
	warnColor.Printf(format, args...)
}

// Warnln prints a warning message in yellow color with a newline.
func Warnln(format string, args ...interface{}) {
	warnColor.Printf(format+"\n", args...)
}

// Error prints an error message in red color.
func Error(format string, args ...interface{}) {
	errorColor.Printf(format, args...)
}

// Errorf prints an error message in red color (alias for Error).
func Errorf(format string, args ...interface{}) {
	errorColor.Printf(format, args...)
}

// Errorln prints an error message in red color with a newline.
func Errorln(format string, args ...interface{}) {
	errorColor.Printf(format+"\n", args...)
}

// Debug prints a debug message in cyan color.
func Debug(format string, args ...interface{}) {
	debugColor.Printf(format, args...)
}

// Debugf prints a debug message in cyan color (alias for Debug).
func Debugf(format string, args ...interface{}) {
	debugColor.Printf(format, args...)
}

// Debugln prints a debug message in cyan color with a newline.
func Debugln(format string, args ...interface{}) {
	debugColor.Printf(format+"\n", args...)
}

// Success prints a success message in bold green color.
func Success(format string, args ...interface{}) {
	successColor.Printf(format, args...)
}

// Successf prints a success message in bold green color (alias for Success).
func Successf(format string, args ...interface{}) {
	successColor.Printf(format, args...)
}

// Successln prints a success message in bold green color with a newline.
func Successln(format string, args ...interface{}) {
	successColor.Printf(format+"\n", args...)
}

// Fatal prints an error message in red color and exits the application with code 1.
func Fatal(format string, args ...interface{}) {
	errorColor.Printf(format+"\n", args...)
	os.Exit(1)
}

// Print prints a message without color formatting.
func Print(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println prints a message without color formatting with a newline.
func Println(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
