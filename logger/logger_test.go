package logger_test

import (
	"bytes"
	"io"
	"log"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/logger"
)

var (
	logLevelRegexp = regexp.MustCompile(`^\[[A-Z]+\]`)
	fpRegexp       = regexp.MustCompile(`trails.*\.go`)
	msgRegexp      = regexp.MustCompile(`"(.*)"\n$`)
)

func TestTrailsLoggerDebug(t *testing.T) {
	// Arrange
	expected := []byte("hello")
	b := new(bytes.Buffer)
	l := newTestLogger(b)
	tl := logger.New(logger.WithLogger(l))

	// Act
	tl.Debug(string(expected), nil)

	// Assert
	require.Nil(t, b.Bytes())

	// Arrange
	tl = logger.New(logger.WithLevel(logger.LogLevelDebug), logger.WithLogger(l))

	// Act
	tl.Debug(string(expected), nil)

	// Assert
	actual := b.Bytes()
	require.Equal(t, []byte("[DEBUG]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])
}

func TestTrailsLoggerError(t *testing.T) {
	// Arrange
	expected := []byte("hello")
	b := new(bytes.Buffer)
	l := newTestLogger(b)
	tl := logger.New(logger.WithLevel(logger.LogLevelFatal), logger.WithLogger(l))

	// Act
	tl.Error(string(expected), nil)

	// Assert
	require.Nil(t, b.Bytes())

	// Arrange
	tl = logger.New(logger.WithLevel(logger.LogLevelError), logger.WithLogger(l))

	// Act
	tl.Error(string(expected), nil)

	// Assert
	actual := b.Bytes()
	require.Equal(t, []byte("[ERROR]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])

	// Arrange
	tl = logger.New(logger.WithLogger(l))

	// Act
	tl.Error(string(expected), nil)

	// Assert
	actual = b.Bytes()
	require.Equal(t, []byte("[ERROR]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])
}

func TestTrailsLoggerFatal(t *testing.T) {
	// Arrange
	expected := []byte("hello")
	b := new(bytes.Buffer)
	l := newTestLogger(b)
	tl := logger.New(logger.WithLogger(l))

	// Act
	tl.Fatal(string(expected), nil)

	// Assert
	actual := b.Bytes()
	require.Equal(t, []byte("[FATAL]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])

	// Arrange
	tl = logger.New(logger.WithLevel(logger.LogLevelFatal), logger.WithLogger(l))

	// Act
	tl.Fatal(string(expected), nil)

	// Assert
	actual = b.Bytes()
	require.Equal(t, []byte("[FATAL]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])
}

func TestTrailsLoggerInfo(t *testing.T) {
	// Arrange
	expected := []byte("hello")
	b := new(bytes.Buffer)
	l := newTestLogger(b)
	tl := logger.New(logger.WithLevel(logger.LogLevelFatal), logger.WithLogger(l))

	// Act
	tl.Info(string(expected), nil)

	// Assert
	require.Nil(t, b.Bytes())

	// Arrange
	tl = logger.New(logger.WithLevel(logger.LogLevelInfo), logger.WithLogger(l))

	// Act
	tl.Info(string(expected), nil)

	// Assert
	actual := b.Bytes()
	require.Equal(t, []byte("[INFO]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])

	// Arrange
	tl = logger.New(logger.WithLogger(l))

	// Act
	tl.Info(string(expected), nil)

	// Assert
	actual = b.Bytes()
	require.Equal(t, []byte("[INFO]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])
}
func TestTrailsLoggerWarn(t *testing.T) {
	// Arrange
	expected := []byte("hello")
	b := new(bytes.Buffer)
	l := newTestLogger(b)
	tl := logger.New(logger.WithLevel(logger.LogLevelFatal), logger.WithLogger(l))

	// Act
	tl.Warn(string(expected), nil)

	// Assert
	require.Nil(t, b.Bytes())

	// Arrange
	tl = logger.New(logger.WithLevel(logger.LogLevelWarn), logger.WithLogger(l))

	// Act
	tl.Warn(string(expected), nil)

	// Assert
	actual := b.Bytes()
	require.Equal(t, []byte("[WARN]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])

	// Arrange
	tl = logger.New(logger.WithLogger(l))

	// Act
	tl.Warn(string(expected), nil)

	// Assert
	actual = b.Bytes()
	require.Equal(t, []byte("[WARN]"), logLevelRegexp.Find(actual))
	require.Equal(t, []byte("trails/logger/logger_test.go"), fpRegexp.Find(actual))
	require.Equal(t, expected, msgRegexp.FindAllSubmatch(actual, 1)[0][1])

}

func newTestLogger(w io.Writer) *log.Logger {
	return log.New(w, "", 0)
}
