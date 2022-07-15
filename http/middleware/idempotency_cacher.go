package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	idempotencyLock sync.Mutex
	_               IdempotencyCacher = make(IdemResMap)
	_               IdempotencyCacher = IdemResRedis{}
)

// An IdempotencyCacher can store responses paired to idempotency keys.
//
// An IdempotencyCacher ought return newly initialized IdemRes
// when a key does not match an existing IdemRes
type IdempotencyCacher interface {
	Get(ctx context.Context, key string) (IdemRes, bool)
	Set(ctx context.Context, key string, idemRes IdemRes)
}

// An IdemResMap stores idempotency key, IdemRes value pairs in a map.
//
// Server restarts reset this map.
// idemResMap ought not be used for production environments.
type IdemResMap map[string]IdemResMapVal

// NewIdemResMap constructs initializes an IdemResMap
// for use in an Idempotency middleware as a cache.
func NewIdemResMap() IdemResMap { return make(IdemResMap) }

// An IdemResMapVal is stored in an IdemResMap,
// wrapping an IdemRes.
type IdemResMapVal struct {
	IdemRes

	at time.Time
}

// Get retrieves the result of the request matching the idempotency key
// much like a regular map.
func (i IdemResMap) Get(ctx context.Context, key string) (IdemRes, bool) {
	if key == "" {
		return IdemRes{}, false
	}

	select {
	case <-ctx.Done():
		return IdemRes{}, false

	default:
		idempotencyLock.Lock()
		defer idempotencyLock.Unlock()

		v, ok := i[key]
		return v.IdemRes, ok
	}
}

// Set overwrites the value paired to key in the map.
//
// For each call to Set, keys older than 24 hours are evicted.
func (i IdemResMap) Set(ctx context.Context, key string, idemRes IdemRes) {
	select {
	case <-ctx.Done():
		return
	default:
		idempotencyLock.Lock()
		defer idempotencyLock.Unlock()

		yesterday := time.Now().AddDate(0, 0, -1)
		for k, v := range i {
			if v.at.Before(yesterday) {
				delete(i, k)
			}
		}

		i[key] = IdemResMapVal{IdemRes: idemRes, at: time.Now()}
	}
}

// An IdemResRedis connects to a Redis backend
// for the purposes of caching idempotent responses.
type IdemResRedis struct {
	client *redis.Client
}

// NewRedisCache constructs an IdemResRedis with the options passed in.
func NewRedisCache(opts *redis.Options) IdemResRedis {
	return IdemResRedis{client: redis.NewClient(opts)}
}

// Get retrieves the *IdemRes paired to key from the connected Redis backend.
func (i IdemResRedis) Get(ctx context.Context, key string) (IdemRes, bool) {
	select {
	case <-ctx.Done():
		return IdemRes{}, false
	default:
		b, err := i.client.Get(ctx, key).Bytes()
		if err != nil {
			return IdemRes{}, false
		}

		ir := new(IdemRes)
		if err := ir.GobDecode(b); err != nil {
			return IdemRes{}, false
		}

		return *ir, true
	}
}

// Set saves the *IdemRes by pairing it to the key in the Redis backend.
func (i IdemResRedis) Set(ctx context.Context, key string, idemRes IdemRes) {
	select {
	case <-ctx.Done():
		return
	default:
		b, err := idemRes.GobEncode()
		if err != nil {
			return
		}
		i.client.Set(ctx, key, b, time.Duration(24*time.Hour))
	}
}
