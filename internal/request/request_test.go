package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data             string
	byteCountPerRead int
	pos              int
}

func (cr *chunkReader) Read(p []byte) (n int, err error) {
	dataLength := len(cr.data)

	if cr.pos >= dataLength {
		return 0, nil
	}

	endIndex := min(cr.pos+cr.byteCountPerRead, dataLength)

	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	// Test: Good GET request line
	reader := &chunkReader{
		data:             "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good GET request line path
	reader = &chunkReader{
		data:             "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good POST request line path
	data := "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\nContent-Length: 22\r\n\r\n{\"flavor\":\"dark mode\"}"
	reader = &chunkReader{
		data:             data,
		byteCountPerRead: len(data),
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Invalid number of parts in request line
	reader = &chunkReader{
		data:             "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Invalid method (out of order) in request line
	reader = &chunkReader{
		data:             "/coffee GET HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 10,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Invalid version in request line
	reader = &chunkReader{
		data:             "/coffee GET HTTP/1.3\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 7,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestRequestHeadersParse(t *testing.T) {
	// Test: Standard Headers
	reader := &chunkReader{
		data:             "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069", r.Headers.Get("Host"))
	assert.Equal(t, "curl/7.81.0", r.Headers.Get("User-Agent"))
	assert.Equal(t, "*/*", r.Headers.Get("Accept"))

	// Test: Duplicate Headers
	reader = &chunkReader{
		data:             "GET / HTTP/1.1\r\nHost: localhost:42069\r\nHost: localhost:42069\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069,localhost:42069", r.Headers.Get("Host"))

	// Test: Case Insensitive Headers
	reader = &chunkReader{
		data:             "GET / HTTP/1.1\r\nHost: localhost:42069\r\nhost: localhost:42069\r\nuser-agent: curl/7.81.0\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069,localhost:42069", r.Headers.Get("HOST"))
	assert.Equal(t, "curl/7.81.0", r.Headers.Get("USER-AGENT"))

	// Test: Empty Headers
	reader = &chunkReader{
		data:             "GET / HTTP/1.1\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Malformed Header
	reader = &chunkReader{
		data:             "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestRequestBodyParse(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"hello world!\n",
		byteCountPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "hello world!\n", string(r.Body))

	// Test: Empty body, 0 reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 0, len(r.Body))

	// Test: Empty body, no reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 0, len(r.Body))

	// Test: No reported content length but body exists
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n" +
			"hello world!\n",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 0, len(r.Body))

	// Test: Body longer than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"partial content partial content partial content partial content partial content partial content partial content partial content partial content",
		byteCountPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
}
