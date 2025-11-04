package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"httpffomtcp.pinglu.dev/internal/headers"
	"httpffomtcp.pinglu.dev/internal/request"
	"httpffomtcp.pinglu.dev/internal/response"
	"httpffomtcp.pinglu.dev/internal/server"
)

const PORT = 42069

func responseBody400() []byte {
	s := `
		<html>
			<head>
				<title>400 Bad Request</title>
			</head>
			<body>
				<h1>Bad Request</h1>
				<p>Your request honestly kinda sucked.</p>
			</body>
		</html>
	`
	return []byte(s)
}

func responseBody500() []byte {
	s := `
		<html>
			<head>
				<title>500 Internal Server Error</title>
			</head>
			<body>
				<h1>Internal Server Error</h1>
				<p>Okay, you know what? This one is on me.</p>
			</body>
		</html>
	`
	return []byte(s)
}

func responseBody200() []byte {
	s := `
		<html>
			<head>
				<title>200 OK</title>
			</head>
			<body>
				<h1>Success!</h1>
				<p>Your request was an absolute banger.</p>
			</body>
		</html>
	`
	return []byte(s)
}

var sumTrailerKey = strings.ToLower("X-Content-SHA256")
var contentLengthTrailerKey = strings.ToLower("X-Content-Length")

func proxyHandler(w *response.Writer) {
	resp, err := http.Get("https://httpbin.org/stream/100")
	if err != nil {
		msg := []byte(err.Error())

		w.WriteStatusLine(response.STATUS_INTERNAL_ERROR)

		h := response.GetDefaultHeaders(len(msg))
		w.WriteHeaders(h)

		w.WriteBody(msg)
		return
	}
	defer resp.Body.Close()

	err = w.WriteStatusLine(response.STATUS_OK)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	h := response.GetDefaultHeaders(0)
	h.Delete("Content-Length")
	h.Set("Transfer-Encoding", "chunked")
	h.Replace("Content-Type", "application/json")
	h.Set("Trailer", sumTrailerKey)
	h.Set("Trailer", contentLengthTrailerKey)
	err = w.WriteHeaders(h)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	var fullBody bytes.Buffer
	buf := make([]byte, 1024)
	done := false
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				done = true
			}
			break
		}

		data := buf[:n]
		_, err = w.WriteChunkedBody(data)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		_, err = fullBody.Write(data)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}
	}

	if done {
		_, err := w.WriteChunkedBodyDone()
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		sum := sha256.Sum256(fullBody.Bytes())
		t := headers.NewHeaders()
		t.Set(sumTrailerKey, fmt.Sprintf("%x", sum))
		t.Set(contentLengthTrailerKey, strconv.Itoa(fullBody.Len()))
		w.WriteTrailers(t)
	}
}

func videoHandler(w *response.Writer) {
	data, err := os.ReadFile("assets/vim.mp4")
	if err != nil {
		msg := []byte(err.Error())
		h := response.GetDefaultHeaders(len(msg))
		err := w.WriteStatusLine(response.STATUS_INTERNAL_ERROR)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		err = w.WriteHeaders(h)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		_, err = w.WriteBody(msg)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}
		return
	}

	h := response.GetDefaultHeaders(len(data))
	h.Replace("content-type", "video/mp4")

	err = w.WriteStatusLine(response.STATUS_OK)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	err = w.WriteHeaders(h)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	_, err = w.WriteBody(data)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}
}

func handler(w *response.Writer, req *request.Request) {
	h := response.GetDefaultHeaders(0)
	statusCode := response.STATUS_OK
	body := responseBody200()

	switch req.RequestLine.RequestTarget {
	case "/httpbin/stream/100":
		proxyHandler(w)
		return
	case "/video":
		videoHandler(w)
		return
	case "/yourproblem":
		statusCode = response.STATUS_BAD_REQUEST
		body = responseBody400()
	case "/myproblem":
		statusCode = response.STATUS_INTERNAL_ERROR
		body = responseBody500()
	}

	h.Replace("Content-Type", "text/html")
	h.Replace("Content-Length", strconv.Itoa(len(body)))

	err := w.WriteStatusLine(statusCode)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	err = w.WriteHeaders(h)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}

	_, err = w.WriteBody(body)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
	}
}

func main() {
	server, err := server.Serve(PORT, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", PORT)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
