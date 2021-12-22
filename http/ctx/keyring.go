package ctx

type Key string

// Key returns key so it can be used as a key a map[string].
func (k Key) Key() string { return string(k) }

// String formats the stringified key with additional contextual information
func (k Key) String() string {
	return "http context key: " + string(k)
}

const _ Key = ""

// Something KeyRingable because it stores arbitrary keys, accessible by a string name,
// and makes it convenient to grab a CurrentUserKey or SessionKey.
type KeyRingable interface {
	CurrentUserKey() CtxKeyable
	Key(name string) CtxKeyable
	SessionKey() CtxKeyable
	keys() map[string]CtxKeyable
}

// KeyRing stores CtxKeyables and cannot be mutated outside of a constructor.
type KeyRing struct {
	session     string
	currentUser string
	internal    map[string]CtxKeyable
}

// NewKeyRing constructs a KeyRing from the given CtxKeyables.
// NewKeyRing requires keys to be retrieved through CurrentUserKey() and SessionKey(), respectively.
// NewKeyRing accepts an arbitrary number of other CtxKeyables, accessible through the Key method.
func NewKeyRing(sessionKey, currentUserKey CtxKeyable, additional ...CtxKeyable) KeyRingable {
	if sessionKey == nil || currentUserKey == nil {
		return nil
	}
	kr := &KeyRing{
		session:     sessionKey.Key(),
		currentUser: currentUserKey.Key(),
		internal: map[string]CtxKeyable{
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

// CurrentUserKey returns the key set in the currentUserKey parameter of NewKeyRing or nil.
func (kr *KeyRing) CurrentUserKey() CtxKeyable {
	return kr.internal[kr.currentUser]
}

// Key returns the key by name (i.e., CtxKeyable.Key()) or nil.
func (kr *KeyRing) Key(name string) CtxKeyable {
	return kr.internal[name]
}

// SessionKey returns the key set in the sessionKey parameter of NewKeyRing or nil.
func (kr *KeyRing) SessionKey() CtxKeyable {
	return kr.internal[kr.session]
}

// keys exposes the internal map of CtxKeyables.
func (kr *KeyRing) keys() map[string]CtxKeyable { return kr.internal }

// WithKeyRing constructs a new KeyRingable from the parent
// and adds additional CtxKeyables to the new KeyRingable.
func WithKeyRing(parent KeyRingable, additional ...CtxKeyable) KeyRingable {
	sk := parent.SessionKey()
	ck := parent.CurrentUserKey()
	kr := &KeyRing{
		session:     sk.Key(),
		currentUser: ck.Key(),
		internal:    make(map[string]CtxKeyable),
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
