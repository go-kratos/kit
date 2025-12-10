package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidToken is returned when a page token is invalid.
var ErrInvalidToken = errors.New("invalid page token")

// TokenOption defines options for the TokenGenerator.
type TokenOption func(*tokenGenerator)

// WithTokenSalt sets a salt for the token generation.
func WithTokenSalt(salt string) TokenOption {
	return func(t *tokenGenerator) {
		t.salt = salt
	}
}

// NewTokenGenerator provides a new instance of a TokenGenerator.
func NewTokenGenerator(opts ...TokenOption) TokenGenerator {
	t := &tokenGenerator{}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// TokenRequest represents a request that contains pagination tokens.
type TokenRequest interface {
	// GetPageToken returns the page token of the request.
	GetPageToken() string
}

// TokenGenerator generates a page token for a given index.
type TokenGenerator interface {
	ForIndex(int) string
	GetIndex(string) (int, error)
}

type tokenGenerator struct {
	salt string
}

// Parse extracts the index from the page token in the request.
func (t *tokenGenerator) Parse(req TokenRequest) (int, error) {
	token := req.GetPageToken()
	return t.GetIndex(token)
}

// ForIndex generates a page token for the given index.
func (t *tokenGenerator) ForIndex(i int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%d", t.salt, i)))
}

// GetIndex retrieves the index from the given page token.
func (t *tokenGenerator) GetIndex(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	bs, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidToken
	}
	if !strings.HasPrefix(string(bs), t.salt) {
		return 0, ErrInvalidToken
	}
	index, err := strconv.Atoi(strings.TrimPrefix(string(bs), t.salt))
	if err != nil {
		return 0, ErrInvalidToken
	}
	return index, nil
}
