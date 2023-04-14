package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"hash"
	"io"
	"net/http"
	"sync"
)

const (
	IdempotencyHeader = "Idempotency-Key"
)

var (
	_            http.ResponseWriter = idemReqWriter{}
	hasherLock                       = sync.Mutex{}
	defaultCache                     = NewIdemResMap()
	defaultHash                      = sha256.New()
)

// Idempotent returns a middleware.Adapter that enables features
// of idempotency on a POST endpoint.
// GET, DELETE, PUT, & PATCH are idempotent by definition.
//
// Idempotent pulls a key (a UUID v4 string) from request headers
// to base the uniqueness of a POST request around.
//
// If a previous request has not used that key,
// Idempotent pairs all of the following values to the key:
// - the body of the request
// - the body of the resulting response
// - the status code of the resulting response
//
// If that key has been used before (and has not expired),
// Idempotent falls into one of these scenarios:
//
//   - if a status code has not been set for that key,
//     Idempotent responds with 409 since the idempotent request is still processing
//
//   - if the newly requested resource (the URI) does not match the original,
//     Idempotent responsds with 422
//
//   - if the new request's body does not match the body of the original request's,
//     Idempotent responds with 422
//
// - Idempotent writes the status code and body set for the key
//
// cache and hasher can be nil.
// Idempotent will use a default cache and implementation of hash.Hash, accordingly.
//
// Idempotent implements the draft Idempotent HTTP Header Field specification:
// https://tools.ietf.org/id/draft-idempotency-header-01.html
func Idempotent(cache IdempotencyCacher, hasher hash.Hash) Adapter {
	if cache == nil {
		cache = defaultCache
	}

	if hasher == nil {
		hasher = defaultHash
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			key := r.Header.Get(IdempotencyHeader)
			if key == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			hasherLock.Lock()
			teeBody := bytes.NewBuffer(nil)
			check := io.TeeReader(r.Body, teeBody)
			if _, err := io.Copy(hasher, check); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = io.NopCloser(teeBody)
			sum := hasher.Sum(nil)
			hasher.Reset()
			hasherLock.Unlock()

			ir, ok := cache.Get(r.Context(), key)
			if ok {
				if ir.Status == 0 {
					w.WriteHeader(http.StatusConflict)
					return
				}

				if ir.URI != r.URL.RequestURI() || bytes.Compare(ir.Req, sum) != 0 {
					w.WriteHeader(http.StatusUnprocessableEntity)
					return
				}

				w.WriteHeader(ir.Status)
				w.Write(ir.Body.Bytes())
				return
			}

			ir = NewIdemRes(r.URL.RequestURI(), sum)
			cache.Set(r.Context(), key, ir)

			irw := idemReqWriter{
				ctx: r.Context(),
				c:   cache,
				i:   &ir,
				k:   key,
				w:   w,
			}
			handler.ServeHTTP(irw, r)
		})
	}
}

// An IdemRes is data from an HTTP response
// that can be reused when another request
// matches the same idempotency key.
type IdemRes struct {
	Body   *bytes.Buffer
	Req    []byte
	Status int
	URI    string
}

// An idemResGob is an intermediate represenation of
// an IdemRest for the purposes of gob encoding/decoding.
//
// idemResGob is necessary as long as pkg gob cannot decode/encode
// fields in an IdemRes (e.g., Body).
type idemResGob struct {
	B []byte
	R []byte
	S int
	U string
}

// NewIdemRes constructs a new *IdemRes.
func NewIdemRes(uri string, hashedBody []byte) IdemRes {
	return IdemRes{Body: bytes.NewBuffer(nil), URI: uri, Req: hashedBody}
}

// GobDecode unmarshals the gob-encoded []byte into fields of the *IdemRes.
//
// GobDecode implements gob.GobDecoder.
func (i *IdemRes) GobDecode(b []byte) error {
	g := new(idemResGob)
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(g); err != nil {
		return err
	}

	i.Body = bytes.NewBuffer(g.B)
	i.Req, i.Status, i.URI = g.R, g.S, g.U
	return nil
}

// GobEncode marshals the fields of the IdemRes into a gob-encoded []byte.
//
// GobEncode implements gob.GobEncoder.
func (i IdemRes) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	g := idemResGob{i.Body.Bytes(), i.Req, i.Status, i.URI}
	if err := gob.NewEncoder(buf).Encode(g); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// An idemReqWriter pairs an IdemRes with an http.ResponseWriter
// so both can be written to by an HTTP handler.
// Changes to the IdemRes in such a way are saved in the cache.
//
// An idemReqWriter implements http.ResponseWriter.
type idemReqWriter struct {
	ctx context.Context
	c   IdempotencyCacher
	i   *IdemRes
	k   string
	w   http.ResponseWriter
}

// Header returns the http.Header of the underlying http.ResponseWriter.
func (irw idemReqWriter) Header() http.Header { return irw.w.Header() }

// Write writes the bytes to all consumers the idemReqWriter is concerned with.
func (irw idemReqWriter) Write(b []byte) (int, error) {
	select {
	case <-irw.ctx.Done():
		return 0, nil
	default:
		if irw.i.Status == 0 {
			irw.WriteHeader(http.StatusOK)
		}

		n, err := irw.w.Write(b)
		if err != nil {
			return n, err
		}

		if _, err = irw.i.Body.Write(b); err != nil {
			return n, err
		}

		irw.c.Set(irw.ctx, irw.k, *irw.i)
		return n, nil
	}
}

// WriteHeader copies the status code about to be written to the *idemReq for later reuse
// before actually writting the status status code.
func (irw idemReqWriter) WriteHeader(s int) {
	select {
	case <-irw.ctx.Done():
		return
	default:
		irw.w.WriteHeader(s)
		irw.i.Status = s
		irw.c.Set(irw.ctx, irw.k, *irw.i)
	}
}
