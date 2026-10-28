package domain

import "errors"

// Errors of the imports domain. The HTTP adapter maps them to status codes.
var (
	ErrImportNotFound     = errors.New("imports: import not found")
	ErrInvalidImport      = errors.New("imports: invalid import data")
	ErrFileTooLarge       = errors.New("imports: file too large")
	ErrUnreadableFile     = errors.New("imports: the file cannot be read")
	ErrImportNotValidated = errors.New("imports: only a validated import can be confirmed")
	ErrImportNotQueued    = errors.New("imports: the import was not confirmed")
	ErrImportNotRunning   = errors.New("imports: the import is not running")
	ErrImportFinished     = errors.New("imports: the import already ended")
)
