package domain

import "errors"

// Errors of the feedback domain; the HTTP adapter maps them to statuses.
var (
	ErrInvalidMessage  = errors.New("feedback: the message must have between 1 and 2000 characters")
	ErrInvalidScreen   = errors.New("feedback: the screen must be a path of up to 200 characters")
	ErrInvalidVersion  = errors.New("feedback: the app version must have between 1 and 50 characters")
	ErrFeedbackMissing = errors.New("feedback: not found")
)
