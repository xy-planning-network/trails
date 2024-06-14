package req

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/xy-planning-network/trails"
)

type Parser struct {
	queryParamDecoder queryParamDecoder
	validator
}

func NewParser() *Parser {
	return &Parser{
		queryParamDecoder: newQueryParamDecoder(),
		validator:         newValidator(),
	}
}

// ParseBody decodes into a pointer to a struct the JSON data in *http.Request.Body.
// If successful, ParseBody runs validation against the contents,
// returning an ErrNotValid if the data fails validation rules.
//
// ParseBody reads the entire r.Body and can't be read from again.
// Use a [io.TeeReader] if r.Body needs to be reused after calling ParseBody.
func (p *Parser) ParseBody(body io.Reader, structPtr any) error {
	var ourFault *json.InvalidUnmarshalError
	err := json.NewDecoder(body).Decode(structPtr)
	if errors.As(err, &ourFault) {
		return fmt.Errorf("trails/http/req: %w: ParseBody called with non-pointer: %s", trails.ErrBadAny, err)
	}

	if err != nil {
		return fmt.Errorf("trails/http/req: %w: failed decoding request body: %s", trails.ErrBadFormat, err)
	}

	if err := p.validate(structPtr); err != nil {
		return fmt.Errorf("trails/http/req: %T failed validation: %w", structPtr, err)
	}

	return nil
}

// ParseQueryParams decodes into a pointer to a struct the query param data in *http.Request.URL.Query.
// If successful, ParseQueryParams runs validation against the contents,
// returning an ErrNotValid if the data fails validation rules.
func (p *Parser) ParseQueryParams(params url.Values, structPtr any) error {
	if err := p.queryParamDecoder.decode(structPtr, params); err != nil {
		return fmt.Errorf("trails/http/req: failed decoding request query params: %w", err)
	}

	if err := p.validate(structPtr); err != nil {
		return fmt.Errorf("trails/http/req: %T failed validation: %w", structPtr, err)
	}

	return nil
}
