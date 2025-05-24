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

func Info(format string, args ...interface{}) {
	infoColor.Printf(format, args...)
}

func Infof(format string, args ...interface{}) {
	infoColor.Printf(format, args...)
}

func Infoln(format string, args ...interface{}) {
	infoColor.Printf(format+"\n", args...)
}

func Warn(format string, args ...interface{}) {
	warnColor.Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	warnColor.Printf(format, args...)
}

func Warnln(format string, args ...interface{}) {
	warnColor.Printf(format+"\n", args...)
}

func Error(format string, args ...interface{}) {
	errorColor.Printf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	errorColor.Printf(format, args...)
}

func Errorln(format string, args ...interface{}) {
	errorColor.Printf(format+"\n", args...)
}

func Debug(format string, args ...interface{}) {
	debugColor.Printf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	debugColor.Printf(format, args...)
}

func Debugln(format string, args ...interface{}) {
	debugColor.Printf(format+"\n", args...)
}

func Success(format string, args ...interface{}) {
	successColor.Printf(format, args...)
}

func Successf(format string, args ...interface{}) {
	successColor.Printf(format, args...)
}

func Successln(format string, args ...interface{}) {
	successColor.Printf(format+"\n", args...)
}

func Fatal(format string, args ...interface{}) {
	errorColor.Printf(format+"\n", args...)
	os.Exit(1)
}

func Print(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func Println(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
