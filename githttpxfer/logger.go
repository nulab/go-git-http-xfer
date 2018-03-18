package githttpxfer

import "log"

type Logger interface {
	Error(args ...interface{})
}

type defaultLogger struct {}

func (*defaultLogger) Error(args ...interface{}) {
	log.Print(args...)
}