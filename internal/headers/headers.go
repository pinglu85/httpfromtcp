package headers

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"unicode"
)

var ERROR_MALFORMED_HEADER = errors.New("malformed header")
var ERROR_INVALID_FIELD_NAME = errors.New("invalid field name")
var CRLF = []byte("\r\n")
var CRLF_LEN = len(CRLF)
var VALID_SPECIAL_CHARS = []rune{'!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~'}

type Headers map[string]string

func NewHeaders() Headers {
	return Headers{}
}

func (h Headers) Get(key string) string {
	lowercasedKey := strings.ToLower(key)
	return h[lowercasedKey]
}

func (h Headers) Set(key, value string) {
	lowercasedKey := strings.ToLower(key)

	if existingValue, found := h[lowercasedKey]; found {
		h[lowercasedKey] = existingValue + "," + value
	} else {
		h[lowercasedKey] = value
	}
}

func (h Headers) Replace(key, value string) {
	lowercasedKey := strings.ToLower(key)
	h[lowercasedKey] = value
}

func (h Headers) Delete(key string) {
	lowercasedKey := strings.ToLower(key)
	delete(h, lowercasedKey)
}

func (h Headers) Parse(data []byte) (int, bool, error) {
	crlfIndex := bytes.Index(data, CRLF)
	if crlfIndex == -1 {
		return 0, false, nil
	}

	if crlfIndex == 0 {
		return CRLF_LEN, true, nil
	}

	s := string(data[:crlfIndex])

	colonIndex := strings.Index(s, ":")
	if colonIndex == -1 {
		return 0, false, ERROR_MALFORMED_HEADER
	}

	key := s[:colonIndex]
	key = strings.TrimLeft(key, " ")
	if !validHeaderKey(key) {
		return 0, false, ERROR_INVALID_FIELD_NAME
	}

	value := s[colonIndex+1:]
	value = strings.TrimSpace(value)

	h.Set(key, value)

	return len(s) + CRLF_LEN, false, nil
}

func validHeaderKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	for _, r := range key {
		if !unicode.IsDigit(r) &&
			(r < 'a' || r > 'z') &&
			(r < 'A' || r > 'Z') &&
			!slices.Contains(VALID_SPECIAL_CHARS, r) {
			return false
		}
	}

	return true
}
