package ctx

import (
	"sort"
)

type CtxKeyable interface {
	// The key as in a key-value pair
	Key() string

	// A stringified version of the key, for logging
	String() string
}

type ByCtxKeyable []CtxKeyable

var _ sort.Interface = ByCtxKeyable([]CtxKeyable{})

func (k ByCtxKeyable) Len() int           { return len(k) }
func (k ByCtxKeyable) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }
func (k ByCtxKeyable) Less(i, j int) bool { return k[i].String() < k[j].String() }
