package server

import (
	"github.com/andrejbaran/apns-ms/apns"
)

var logger apns.LoggerInterface = new(nullLogger)

// SetLogger sets the package logger
func SetLogger(l apns.LoggerInterface) {
	logger = l
}

type nullLogger struct {
}

func (l *nullLogger) Println(args ...interface{})               {}
func (l *nullLogger) Printf(format string, args ...interface{}) {}
func (l *nullLogger) Print(args ...interface{})                 {}

func (l *nullLogger) Panicf(format string, args ...interface{}) {}
func (l *nullLogger) Panic(args ...interface{})                 {}

func (l *nullLogger) Fatalf(format string, args ...interface{}) {}
func (l *nullLogger) Fatal(args ...interface{})                 {}

func (l *nullLogger) Errorf(format string, args ...interface{}) {}
func (l *nullLogger) Error(entries ...interface{})              {}

func (l *nullLogger) Warningf(format string, args ...interface{}) {}
func (l *nullLogger) Warning(entries ...interface{})              {}

func (l *nullLogger) Noticef(format string, args ...interface{}) {}
func (l *nullLogger) Notice(entries ...interface{})              {}

func (l *nullLogger) Infof(format string, args ...interface{}) {}
func (l *nullLogger) Info(entries ...interface{})              {}

func (l *nullLogger) Debugf(format string, args ...interface{}) {}
func (l *nullLogger) Debug(entries ...interface{})              {}
