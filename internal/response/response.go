package response

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"httpffomtcp.pinglu.dev/internal/headers"
)

type StatusCode int

const (
	STATUS_OK             StatusCode = 200
	STATUS_BAD_REQUEST    StatusCode = 400
	STATUS_INTERNAL_ERROR StatusCode = 500
)

type ReasonPhrase string

const (
	REASON_OK             ReasonPhrase = "OK"
	REASON_BAD_REQUEST    ReasonPhrase = "Bad Request"
	REASON_INTERNAL_ERROR ReasonPhrase = "Interval Server Error"
)

const HTTP_VERSION = "HTTP/1.1"
const CRLF = "\r\n"

var ZERO_CRLF = []byte("0\r\n")

type WriterState string

const (
	INITIALIZED      WriterState = "initialized"
	STATUS_LINE_DONE WriterState = "status line done"
	HEADERS          WriterState = "headers"
	BODY             WriterState = "body"
	BODY_DONE        WriterState = "body done"
	TRAILERS         WriterState = "trailers"
)

var ERROR_WRONG_WRITE_ORDER = errors.New("WriteStatusLine, WriteHeaders, and WriteBody should be called in the correct order.")

type Writer struct {
	writerState WriterState
	writer      io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writerState: INITIALIZED,
		writer:      w,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.writerState != INITIALIZED {
		return ERROR_WRONG_WRITE_ORDER
	}

	var reason ReasonPhrase

	switch statusCode {
	case STATUS_OK:
		reason = REASON_OK
	case STATUS_BAD_REQUEST:
		reason = REASON_BAD_REQUEST
	case STATUS_INTERNAL_ERROR:
		reason = REASON_INTERNAL_ERROR
	default:
		reason = ""
	}

	statusLine := fmt.Sprintf("%s %d %s%s", HTTP_VERSION, statusCode, reason, CRLF)
	_, err := w.writer.Write([]byte(statusLine))
	if err != nil {
		return err
	}

	w.writerState = STATUS_LINE_DONE

	return nil
}

func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.writerState != STATUS_LINE_DONE && w.writerState != HEADERS {
		return ERROR_WRONG_WRITE_ORDER
	}
	w.writerState = HEADERS

	return w.writeHeadersImpl(h)
}

func (w *Writer) WriteBody(body []byte) (int, error) {
	if w.writerState != HEADERS && w.writerState != BODY {
		return 0, ERROR_WRONG_WRITE_ORDER
	}
	w.writerState = BODY

	n, err := w.writer.Write(body)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (w *Writer) WriteChunkedBody(body []byte) (int, error) {
	if w.writerState != HEADERS && w.writerState != BODY {
		return 0, ERROR_WRONG_WRITE_ORDER
	}
	w.writerState = BODY

	bodyLen := len(body)
	s := fmt.Sprintf("%X%s", bodyLen, CRLF)
	c := slices.Concat([]byte(s), body, []byte(CRLF))

	n, err := w.writer.Write(c)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.writerState != BODY {
		return 0, ERROR_WRONG_WRITE_ORDER
	}

	c := slices.Concat(ZERO_CRLF, []byte(CRLF))

	n, err := w.writer.Write(c)
	if err != nil {
		return 0, err
	}

	w.writerState = BODY_DONE
	return n, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.writerState != BODY_DONE && w.writerState != TRAILERS {
		return ERROR_WRONG_WRITE_ORDER
	}
	w.writerState = TRAILERS

	return w.writeHeadersImpl(h)
}

func (w *Writer) writeHeadersImpl(h headers.Headers) error {
	b := []byte{}

	for key, value := range h {
		s := fmt.Sprintf("%s: %s%s", key, value, CRLF)
		b = fmt.Append(b, s)
	}

	b = fmt.Append(b, CRLF)

	_, err := w.writer.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()

	h.Set("Content-Length", strconv.Itoa(contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")

	return h
}
