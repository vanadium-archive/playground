// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package log exports three diferent loggers: ErrorLogger, WarnLogger, and
// DebugLogger, and convenience methods for logging messages to different
// loggers depending on the level.
//
// By default, the loggers will only write to stdout. The default logger will
// also be configured to print to stdout rather than stderr.
//
// Calling "InitSyslogLoggers", will create new loggers which will log to both
// syslog and stdout, and also set the default logger to log to syslog and
// stdout. The function will panic if syslog is unavailable.

package log

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
)

func init() {
	// By default, the loggers only log to stdout.
	ErrorLogger = newLogger(os.Stdout, "ERROR: ")
	WarnLogger = newLogger(os.Stdout, "WARN: ")
	DebugLogger = newLogger(os.Stdout, "")

	// Default logger should also log to stdout.
	log.SetOutput(os.Stdout)
}

// InitSyslogLoggers creates loggers that will log to syslog as well as stdout.
// It will panic if syslog is unavailable.
func InitSyslogLoggers() {
	ErrorLogger = newLogger(newSyslogStdoutWriter(syslog.LOG_ERR), "ERROR: ")
	WarnLogger = newLogger(newSyslogStdoutWriter(syslog.LOG_WARNING), "WARN: ")
	DebugLogger = newLogger(newSyslogStdoutWriter(syslog.LOG_DEBUG), "")

	// Default logger should also log to syslog and stdout.
	log.SetOutput(newSyslogStdoutWriter(syslog.LOG_WARNING))
}

var (
	ErrorLogger *log.Logger
	WarnLogger  *log.Logger
	DebugLogger *log.Logger
)

// Debug functions use DebugLogger.
func Debug(args ...interface{}) {
	DebugLogger.Print(args...)
}

func Debugf(s string, args ...interface{}) {
	DebugLogger.Printf(s, args...)
}

// Warn functions use WarnLogger.
func Warn(args ...interface{}) {
	WarnLogger.Print(args...)
}

func Warnf(s string, args ...interface{}) {
	WarnLogger.Printf(s, args...)
}

// Error functions use ErrorLogger.
func Error(args ...interface{}) {
	ErrorLogger.Print(args...)
}

func Errorf(s string, args ...interface{}) {
	ErrorLogger.Printf(s, args...)
}

func Panic(args ...interface{}) {
	ErrorLogger.Panic(args...)
}

func Panicf(s string, args ...interface{}) {
	ErrorLogger.Panicf(s, args...)
}

// Helper method to create a logger with given writer.
func newLogger(w io.Writer, prefix string) *log.Logger {
	return log.New(w, prefix, log.LstdFlags)
}

// Helper method to create a writer that writes to syslog and stdout.
func newSyslogStdoutWriter(level syslog.Priority) io.Writer {
	if syslogWriter, err := syslog.New(level|syslog.LOG_USER, "playground"); err != nil {
		panic(fmt.Errorf("Error connecting to syslog: %v", err))
	} else {
		return io.MultiWriter(syslogWriter, os.Stdout)
	}
}
