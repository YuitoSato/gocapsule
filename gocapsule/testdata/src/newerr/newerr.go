package newerr

import "errors"

// Store is constructed by a plain New() that also returns an error
type Store struct { // want Store:`&\{New\}`
	Path string
}

// New creates a new Store
func New(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}
	return &Store{Path: path}, nil
}

// NewFromEnv creates a Store from the environment; New is still preferred in diagnostics
func NewFromEnv() (*Store, error) {
	return New("/env")
}
