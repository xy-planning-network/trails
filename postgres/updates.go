package postgres

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/xy-planning-network/trails"
	"gorm.io/datatypes"
)

// An Updates is a map of key-value pairs where key is the database column and the value is the data.
type Updates map[string]any

func (u Updates) valid() error {
	if len(u) == 0 {
		return fmt.Errorf("%w: no columns set", trails.ErrMissingData)
	}

	return nil
}

// StripNils removes all entries from the map where the value resolves to nil, i.e. NULL.
func (u Updates) StripNils() {
	for k, v := range u {
		switch t := v.(type) {
		case nil:
			delete(u, k)

		case datatypes.JSON:
			if t == nil || bytes.Equal([]byte(t), []byte(datatypes.JSON(json.RawMessage(`null`)))) {
				delete(u, k)
			}

		case driver.Valuer:
			val, err := t.Value()
			if err != nil || val == nil {
				delete(u, k)
			}

		case trails.Enumerable:
			if err := t.Valid(); err != nil {
				delete(u, k)
			}
		}
	}
}
