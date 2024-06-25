package req

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	v10 "github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	"github.com/xy-planning-network/trails"
)

const (
	failedValidationTmpl = "trails/http/req: %T failed validation: %w"
)

type Parser struct {
	queryParamDecoder *schema.Decoder
	validator         *v10.Validate
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
	err := json.NewDecoder(body).Decode(structPtr)

	var ourFault *json.InvalidUnmarshalError
	if errors.As(err, &ourFault) {
		return fmt.Errorf("trails/http/req: %w: ParseBody called with non-pointer", trails.ErrBadAny)
	}

	if err != nil {
		return fmt.Errorf("trails/http/req: %w: failed decoding request body: %s", trails.ErrBadFormat, err)
	}

	err = p.validator.Struct(structPtr)

	var verrs v10.ValidationErrors
	if errors.As(err, &verrs) {
		err = translateValidationErrors(verrs)
		return fmt.Errorf(failedValidationTmpl, structPtr, err)
	}

	if err != nil {
		// NOTE(dlk): return raw v10.Validator errors until use cases show up for parsing these differently.
		return err
	}

	return nil
}

// ParseQueryParams decodes into a pointer to a struct the query param data in *http.Request.URL.Query.
// If successful, ParseQueryParams runs validation against the contents,
// returning an ErrNotValid if the data fails validation rules.
func (p *Parser) ParseQueryParams(params url.Values, structPtr any) error {
	err := p.queryParamDecoder.Decode(structPtr, params)
	if err != nil {
		if err.Error() == "schema: interface must be a pointer to struct" {
			err = fmt.Errorf("trails/http/req: %w: ParseQueryParams called with non-pointer", trails.ErrBadAny)

			return err
		}

		err = translateDecoderError(err)
		if errors.Is(err, trails.ErrNotValid) {
			return fmt.Errorf(failedValidationTmpl, structPtr, err)
		}

		return fmt.Errorf("trails/http/req: failed decoding request query params: %w", err)
	}

	err = p.validator.Struct(structPtr)

	var verrs v10.ValidationErrors
	if errors.As(err, &verrs) {
		err = translateValidationErrors(verrs)
		return fmt.Errorf(failedValidationTmpl, structPtr, err)
	}

	if err != nil {
		// NOTE(dlk): return raw v10.Validator errors until use cases show up for parsing these differently.
		return err
	}

	return nil
}
