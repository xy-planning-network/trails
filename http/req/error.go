package req

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xy-planning-network/trails"
)

// A ValidationError is an issue with a concrete value not matching the rule set on its field.
type ValidationError struct {
	Field string `json:"field"`
	Got   any    `json:"got"`
	Rule  string `json:"rule,omitempty"`
}

// ValidationErrors is a set of ValidationError.
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var msgs []string
	for _, err := range v {
		msg := fmt.Sprintf("field=%q rule=%q got=%q", err.Field, err.Rule, fmt.Sprint(err.Got))
		msgs = append(msgs, msg)
	}

	return strings.Join(msgs, "\n")
}

func (v ValidationErrors) MarshalJSON() ([]byte, error) {
	var errs struct {
		E []ValidationError `json:"validationErrors,omitempty"`
	}

	for _, err := range v {
		errs.E = append(errs.E, err)
	}

	return json.Marshal(errs)
}

func (ValidationErrors) Unwrap() error { return trails.ErrNotValid }
