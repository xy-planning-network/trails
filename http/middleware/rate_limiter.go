package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// A Visitor tracks a rate limiter and last seen time.
type Visitor struct {
	LastSeen time.Time
	Limiter  *rate.Limiter
}

// A Visitors maps a Visitor to an IP address.
type Visitors struct {
	val map[string]Visitor
	sync.Mutex
}

func NewVisitors() *Visitors { return &Visitors{val: make(map[string]Visitor)} }

// Fetch retrieves the Visitor for the given ip creating a new Visitor if not seen.
//
// Newly created visitors are limited to 5 requests every second with bursts of up to 20.
func (vs *Visitors) Fetch(ip string) Visitor {
	vs.Lock()
	defer vs.Unlock()

	v, ok := vs.val[ip]
	if !ok {
		v = Visitor{Limiter: rate.NewLimiter(5, 20)}
	}

	v.LastSeen = time.Now().UTC()
	vs.val[ip] = v
	return v
}

// cleanup deletes a Visitor from Visitors if they have not been seen in over an hour.
func (vs *Visitors) cleanup() {
	vs.Lock()
	defer vs.Unlock()
	for ip, v := range vs.val {
		if time.Since(v.LastSeen) > 60*time.Minute {
			delete(vs.val, ip)
		}
	}
}

// RateLimit encloses the Visitors map and serves the http.Handler
//
// NOTE: implementation found here:
// https://www.alexedwards.net/blog/how-to-rate-limit-http-requests
//
// If we need anything more sophisticated, https://github.com/didip/tollbooth is
// likely a better option.
func RateLimit(visitors *Visitors) Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if visitors.Fetch(GetIPAddress(r.Header)).Limiter.Allow() {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
				return
			}

			visitors.cleanup()
			h.ServeHTTP(w, r)
		})
	}
}
