package trails

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// A NullTime is a time.Time type formatted as RFC 3339 when serialzed to JSON or SQL and can be nil.
type NullTime struct {
	sql.NullTime
}

// NewNullTime casts a time.Time into a sql.NullTime.
func NewNullTime(t time.Time) NullTime {
	return NullTime{NullTime: sql.NullTime{Time: t, Valid: !t.IsZero()}}
}

// MarshalJSON encodes a NullTime into a JSON-encoded byte slice after checking whether it is valid.
//
// MarshalJSON implements json.Marshaler.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(nt.String())
}

// UnmarshalJSON decodes JSON data into the NullTime pointer.
//
// UnmarshalJSON implements json.Unmarshaler.
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	nt.Valid = false
	if string(b) == "null" || string(b) == "" {
		return nil
	}

	err := nt.Time.UnmarshalJSON(b)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	nt.Valid = true
	return nil
}

func (nt NullTime) String() string {
	if !nt.Valid {
		return ""
	}

	return nt.Time.Format(time.RFC3339)
}
