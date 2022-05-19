package keyring

import (
	"sort"
)

type Keyable interface {
	// The key as in a key-value pair
	Key() string

	// A stringified version of the key, for logging
	String() string
}

type ByKeyable []Keyable

var _ sort.Interface = ByKeyable([]Keyable{})

func (k ByKeyable) Len() int           { return len(k) }
func (k ByKeyable) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }
func (k ByKeyable) Less(i, j int) bool { return k[i].String() < k[j].String() }

type Key string

// Key returns key so it can be used as a key a map[string].
func (k Key) Key() string { return string(k) }

// String formats the stringified key with additional contextual information
func (k Key) String() string {
	return "http context key: " + string(k)
}

const _ Key = ""

// Something Keyringable because it stores arbitrary keys, accessible by a string name,
// and makes it convenient to grab a CurrentUserKey or SessionKey.
type Keyringable interface {
	CurrentUserKey() Keyable
	Key(name string) Keyable
	SessionKey() Keyable
	keys() map[string]Keyable
}

// Keyring stores CtxKeyables and cannot be mutated outside of a constructor.
type Keyring struct {
	session     string
	currentUser string
	internal    map[string]Keyable
}

// NewKeyring constructs a Keyring from the given CtxKeyables.
// NewKeyring requires keys to be retrieved through CurrentUserKey() and SessionKey(), respectively.
// NewKeyring accepts an arbitrary number of other CtxKeyables, accessible through the Key method.
func NewKeyring(sessionKey, currentUserKey Keyable, additional ...Keyable) Keyringable {
	if sessionKey == nil || currentUserKey == nil {
		return nil
	}
	kr := &Keyring{
		session:     sessionKey.Key(),
		currentUser: currentUserKey.Key(),
		internal: map[string]Keyable{
			sessionKey.Key():     sessionKey,
			currentUserKey.Key(): currentUserKey,
		},
	}

	for _, k := range additional {
		if k == nil {
			continue
		}
		kr.internal[k.Key()] = k
	}

	return kr
}

// CurrentUserKey returns the key set in the currentUserKey parameter of NewKeyring or nil.
func (kr *Keyring) CurrentUserKey() Keyable {
	return kr.internal[kr.currentUser]
}

// Key returns the key by name (i.e., CtxKeyable.Key()) or nil.
func (kr *Keyring) Key(name string) Keyable {
	return kr.internal[name]
}

// SessionKey returns the key set in the sessionKey parameter of NewKeyring or nil.
func (kr *Keyring) SessionKey() Keyable {
	return kr.internal[kr.session]
}

// keys exposes the internal map of CtxKeyables.
func (kr *Keyring) keys() map[string]Keyable { return kr.internal }

// WithKeyring constructs a new Keyringable from the parent
// and adds additional CtxKeyables to the new Keyringable.
func WithKeyring(parent Keyringable, additional ...Keyable) Keyringable {
	sk := parent.SessionKey()
	ck := parent.CurrentUserKey()
	kr := &Keyring{
		session:     sk.Key(),
		currentUser: ck.Key(),
		internal:    make(map[string]Keyable),
	}

	for k, v := range parent.keys() {
		kr.internal[k] = v
	}

	for _, k := range additional {
		if k == nil {
			continue
		}

		kr.internal[k.Key()] = k
	}

	return kr
}
