package session

import (
	"net/http"
)

const (
	// Default Flash Type
	FlashError   = "error"
	FlashInfo    = "info"
	FlashSuccess = "success"
	FlashWarning = "warning"

	// Default Flash Msg
	BadCredsMsg      = "Hmm... check those credentials."
	BadInputMsg      = "Hmm... check your form, something isn't correct."
	DefaultErrMsg    = "Uh oh! We've run into an issue."
	EmailNotValidMsg = "It looks like your primary email has not been validated. Please complete this process and try again."
	LinkSentMsg      = "Email sent! Please open the link in your email to reset your password."
	NoAccessMsg      = "Oops, sending you back somewhere safe."
)

var ContactUsErr = DefaultErrMsg + " Please contact us at %s if the issue persists."

// FlashSessionable defines the behavior of a session that includes flashes in it.
type FlashSessionable interface {
	ClearFlashes(w http.ResponseWriter, r *http.Request)
	Flashes(w http.ResponseWriter, r *http.Request) []Flash
	SetFlash(w http.ResponseWriter, r *http.Request, flash Flash) error
}

// A Flash is a structured message set in a session.
type Flash struct {
	Type string `json:"type"`
	Msg  string `json:"message"`
}

func (f Flash) GetClass() string {
	switch f.Type {
	case FlashError:
		return "border-red-500"
	case FlashInfo:
		return "border-blue-500"
	case FlashSuccess:
		return "border-green-500"
	case FlashWarning:
		return "border-orange-500"
	default:
		return "border-gray-500"
	}
}
