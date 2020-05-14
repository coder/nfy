package clog

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

func prnt(c color.Attribute, level string, msg string, args ...interface{}) {
	color.New(c).Fprintf(os.Stderr, level+" ")
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func Debug(msg string, args ...interface{}) {
	if os.Getenv("DEBUG") == "" {
		return
	}
	prnt(color.FgWhite, "debug", msg, args...)
}

func Info(msg string, args ...interface{}) {
	prnt(color.FgBlue, "info", msg, args...)
}

func Success(msg string, args ...interface{}) {
	prnt(color.FgGreen, "success", msg, args...)
}

func Warn(msg string, args ...interface{}) {
	prnt(color.FgYellow, "warn", msg, args...)
}

func Error(msg string, args ...interface{}) {
	prnt(color.FgRed, "error", msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	Error(msg, args...)
	os.Exit(1)
}
