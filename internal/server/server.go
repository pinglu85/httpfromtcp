package server

import (
	"fmt"
	"log"
	"net"

	"httpffomtcp.pinglu.dev/internal/request"
	"httpffomtcp.pinglu.dev/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	closed   bool
	listener net.Listener
	handler  Handler
}

func (s *Server) Close() error {
	s.closed = true
	return s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()

		if s.closed {
			return
		}

		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	w := response.NewWriter(conn)

	req, err := request.RequestFromReader(conn)
	if err != nil {
		w.WriteStatusLine(response.STATUS_BAD_REQUEST)

		msg := []byte(err.Error())

		h := response.GetDefaultHeaders(len(msg))
		w.WriteHeaders(h)

		w.WriteBody(msg)
		return
	}

	s.handler(w, req)
}

func Serve(port uint16, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s := &Server{
		closed:   false,
		listener: listener,
		handler:  handler,
	}

	// Listen for requests in the background
	go s.listen()

	return s, nil
}
