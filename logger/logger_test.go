package logger_test

import (
	"io"
	"log"
	"regexp"
)

var (
	logLevelRegexp = regexp.MustCompile(`^\[[A-Z]+\]`)
	fpRegexp       = regexp.MustCompile(`trails.*\.go`)
	msgRegexp      = regexp.MustCompile(`"(.*)"\n$`)
)

func newTestLogger(w io.Writer) *log.Logger {
	return log.New(w, "", 0)
}
