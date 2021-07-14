package ctx

// Something KeyRingable because it stores arbitrary keys, accessible by a string name,
// and makes it convenient to grab a CurrentUserKey or SessionKey.
type KeyRingable interface {
	CurrentUserKey() CtxKeyable
	Key(name string) CtxKeyable
	SessionKey() CtxKeyable
}

// KeyRing stores CtxKeyables and cannot be mutated outside of a constructor.
type KeyRing struct {
	session     string
	currentUser string
	keys        map[string]CtxKeyable
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
		keys: map[string]CtxKeyable{
			sessionKey.Key():     sessionKey,
			currentUserKey.Key(): currentUserKey,
		},
	}

	for _, k := range additional {
		if k == nil {
			continue
		}
		kr.keys[k.Key()] = k
	}

	return kr
}

// CurrentUserKey returns the key set in the currentUserKey parameter of NewKeyRing or nil.
func (kr *KeyRing) CurrentUserKey() CtxKeyable {
	return kr.keys[kr.currentUser]
}

// Key returns the key by name (i.e., CtxKeyable.Key()) or nil.
func (kr *KeyRing) Key(name string) CtxKeyable {
	return kr.keys[name]
}

// SessionKey returns the key set in the sessionKey parameter of NewKeyRing or nil.
func (kr *KeyRing) SessionKey() CtxKeyable {
	return kr.keys[kr.session]
}
