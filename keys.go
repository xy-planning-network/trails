package trails

import "sort"

type Key string

const (
	CurrentUserKey Key = "CurrentUserKey"
	IpAddrKey      Key = "IpAddrKey"
	RequestIDKey   Key = "RequestIDKey"
	SessionKey     Key = "SessionKey"
)

// String formats the stringified key with additional contextual information
func (k Key) String() string {
	return "trails context key: " + string(k)
}

type ByKey []Key

var _ sort.Interface = ByKey([]Key{})

func (k ByKey) Len() int           { return len(k) }
func (k ByKey) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }
func (k ByKey) Less(i, j int) bool { return string(k[i]) < string(k[j]) }

// UniqueSort sorts, uniques and removes zero keys.
func (k ByKey) UniqueSort() ByKey {
	sort.Sort(k)

	// filter cribbed from: https://github.com/golang/go/wiki/SliceTricks#in-place-deduplicate-comparable
	j := 0
	for i := 0; i < len(k); i++ {
		isNotZero := string(k[j]) != ""
		isNotPrev := string(k[j]) != string(k[i])

		if isNotPrev && isNotZero {
			j++
		}

		if isNotPrev {
			k[j] = k[i]
		}
	}

	none := len(k) == 0
	empty := !none && string(k[0]) == ""
	if none || empty {
		return ByKey{}
	}

	return k[:j+1]
}
