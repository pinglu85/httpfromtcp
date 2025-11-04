package headers

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T) {
	crlf := []byte("\r\n")

	// Test: Valid single header
	h := NewHeaders()
	data1 := []byte("Host: localhost:42069\r\n")
	data := slices.Concat(data1, crlf)
	n, done, err := h.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "localhost:42069", h.Get("Host"))
	assert.Equal(t, len(data1), n)
	assert.False(t, done)

	// Test: Valid single header with extra whitespace
	h = NewHeaders()
	data1 = []byte("       Host:    localhost:42069      \r\n")
	data = slices.Concat(data1, crlf)
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "localhost:42069", h.Get("Host"))
	assert.Equal(t, len(data1), n)
	assert.False(t, done)

	// Test: Case insensitive header
	h = NewHeaders()
	data1 = []byte("       host:    localhost:42069      \r\n")
	data = slices.Concat(data1, crlf)
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "localhost:42069", h.Get("Host"))
	assert.Equal(t, len(data1), n)
	assert.False(t, done)

	// Test: Valid 2 headers with existing headers
	h = NewHeaders()
	data1 = []byte("       Set-Person': prime-loves-zig     \r\n")
	data2 := []byte("   Set-Person': lane-loves-go \r\n")
	data = slices.Concat(data1, data2, crlf)
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "prime-loves-zig", h.Get("set-person'"))
	assert.Equal(t, len(data1), n)
	assert.False(t, done)
	n, done, err = h.Parse(data[n:])
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "prime-loves-zig,lane-loves-go", h.Get("set-person'"))
	assert.Equal(t, len(data2), n)
	assert.False(t, done)

	// Test: Valid done
	h = NewHeaders()
	data = []byte("\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, len(crlf), n)
	assert.True(t, done)

	// Test: Invalid spacing header
	h = NewHeaders()
	data = []byte("       Host : localhost:42069       \r\n\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid character in header key
	h = NewHeaders()
	data = []byte("H©st:localhost:42069\r\n\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}
