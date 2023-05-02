package middleware

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xy-planning-network/trails"
	"golang.org/x/exp/slog"
)

const (
	passwordParam     = "password"
	contentLenHeader  = "Content-Length"
	contentTypeHeader = "Content-Type"
	referrerHeader    = "Referrer"
	userAgentHeader   = "User-Agent"
)

// LogRequest logs the a LogRequestRecord using the provided handler.
//
// For the LogRequestRecord.URI, LogRequest masks query params matching these keys with trails.LogMaskVal:
// - password
//
// If handler is nil, NoopAdapter returns and this middleware does nothing.
func LogRequest(ls *slog.Logger) Adapter {
	if ls == nil {
		return NoopAdapter
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			writer := &requestLogger{ResponseWriter: w, status: http.StatusOK}
			h.ServeHTTP(writer, r)

			end := time.Since(start).Milliseconds()

			rec := newRecord(writer, r)
			rec.Duration = end

			var msg string // NOTE(dlk): no message for now.

			// NOTE(dlk): LogAttrs is not the standard method to use,
			// but we know we only have a []slog.Attr to report,
			// so let's leverage it.
			//
			// TODO(dlk): using r.Context() may not be appropriate.
			// Unfortunately, outside of configuring a slog.Handler to pull
			// values from a context.Context into slog.Attr,
			// contexts appear to not be used by slog.
			// This includes not watching for cancellations.
			// The ranger.Context may be the best fit for this.
			// We'll reassess when slog makes into a Go version.
			ls.LogAttrs(r.Context(), slog.LevelInfo, msg, rec.attrs()...)
		})
	}
}

// A LogRequestRecord represents the fields that a LogRequest
type LogRequestRecord struct {
	BodySize       int    `json:"bodySize"`
	Duration       int64  `json:"duration"`
	Host           string `json:"host"`
	ID             string `json:"id"`
	IPAddr         string `json:"remoteAddr"`
	Method         string `json:"method"`
	Path           string `json:"path"`
	Protocol       string `json:"protocol"`
	Referrer       string `json:"referrer"`
	ReqContentLen  int    `json:"contentLength"`
	ReqContentType string `json:"contentType"`
	Scheme         string `json:"scheme"`
	Status         int    `json:"status"`
	URI            string `json:"uri"`
	UserAgent      string `json:"userAgent"`
}

// newRecord constructs a record from the values availabe in w & r.
func newRecord(w *requestLogger, r *http.Request) LogRequestRecord {
	// TODO(dlk): if there's a compelling reason for constructing a LogRequestRecord
	// outside this package, this constructor and LogRequestRecord.attrs could be exported.
	uri := new(url.URL)
	*uri = *r.URL
	q := r.URL.Query()
	mask(q, passwordParam)
	uri.RawQuery = q.Encode()

	contLen, _ := strconv.Atoi(r.Header.Get(contentLenHeader))
	id, _ := r.Context().Value(trails.RequestIDKey).(string)
	ip, _ := r.Context().Value(trails.IpAddrKey).(string)

	return LogRequestRecord{
		BodySize:       w.bodySize,
		Host:           r.Host,
		ID:             id,
		IPAddr:         ip,
		Method:         r.Method,
		Path:           r.URL.Path,
		Protocol:       r.Proto,
		Referrer:       r.Header.Get(referrerHeader),
		ReqContentLen:  contLen,
		ReqContentType: r.Header.Get(contentTypeHeader),
		Scheme:         r.URL.Scheme,
		Status:         w.status,
		URI:            uri.RequestURI(),
		UserAgent:      r.Header.Get(userAgentHeader),
	}
}

func (r LogRequestRecord) attrs() []slog.Attr {
	return []slog.Attr{
		slog.Int64("duration", r.Duration),
		slog.String("host", r.Host),
		slog.String("id", r.ID),
		slog.String("remoteAddr", r.IPAddr),
		slog.String("method", r.Method),
		slog.String("path", r.Path),
		slog.String("protocol", r.Protocol),
		slog.String("referrer", r.Referrer),
		slog.Int("contentLength", r.ReqContentLen),
		slog.String("contentType", r.ReqContentType),
		slog.Int("bodySize", r.BodySize),
		slog.String("scheme", r.Scheme),
		slog.Int("status", r.Status),
		slog.String("uri", r.URI),
		slog.String("userAgent", r.UserAgent),
	}
}

type requestLogger struct {
	http.ResponseWriter
	status   int
	bodySize int
}

func (rl *requestLogger) Header() http.Header { return rl.ResponseWriter.Header() }

// Unwrap exposes the underlying http.ResponseWriter.
//
// NOTE(dlk): include in order to simplify using [net/http.ResponseController].
func (rl *requestLogger) Unwrap() http.ResponseWriter { return rl.ResponseWriter }
func (rl *requestLogger) Write(b []byte) (int, error) {
	size, err := rl.ResponseWriter.Write(b)
	rl.bodySize += size

	return size, err

}

func (rl *requestLogger) WriteHeader(code int) {
	rl.status = code
	rl.ResponseWriter.WriteHeader(code)
}

// mask replaces all instances of key in q with trails.LogMaskVal.
func mask(q url.Values, key string) {
	if val := q.Get(key); val != "" {
		q.Set(key, trails.LogMaskVal)
	}
}
