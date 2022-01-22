package ctx

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
	CurrentUserKey() CtxKeyable
	Key(name string) CtxKeyable
	SessionKey() CtxKeyable
	keys() map[string]CtxKeyable
}

// Keyring stores CtxKeyables and cannot be mutated outside of a constructor.
type Keyring struct {
	session     string
	currentUser string
	internal    map[string]CtxKeyable
}

// NewKeyring constructs a Keyring from the given CtxKeyables.
// NewKeyring requires keys to be retrieved through CurrentUserKey() and SessionKey(), respectively.
// NewKeyring accepts an arbitrary number of other CtxKeyables, accessible through the Key method.
func NewKeyring(sessionKey, currentUserKey CtxKeyable, additional ...CtxKeyable) Keyringable {
	if sessionKey == nil || currentUserKey == nil {
		return nil
	}
	kr := &Keyring{
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

// CurrentUserKey returns the key set in the currentUserKey parameter of NewKeyring or nil.
func (kr *Keyring) CurrentUserKey() CtxKeyable {
	return kr.internal[kr.currentUser]
}

// Key returns the key by name (i.e., CtxKeyable.Key()) or nil.
func (kr *Keyring) Key(name string) CtxKeyable {
	return kr.internal[name]
}

// SessionKey returns the key set in the sessionKey parameter of NewKeyring or nil.
func (kr *Keyring) SessionKey() CtxKeyable {
	return kr.internal[kr.session]
}

// keys exposes the internal map of CtxKeyables.
func (kr *Keyring) keys() map[string]CtxKeyable { return kr.internal }

// WithKeyring constructs a new Keyringable from the parent
// and adds additional CtxKeyables to the new Keyringable.
func WithKeyring(parent Keyringable, additional ...CtxKeyable) Keyringable {
	sk := parent.SessionKey()
	ck := parent.CurrentUserKey()
	kr := &Keyring{
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
