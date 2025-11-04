package main

import (
	"fmt"
	"log"
	"net"

	"httpffomtcp.pinglu.dev/internal/request"
)

// func getLinesFromReader(f io.ReadCloser) <-chan string {
// 	// Create a buffered channel of strings that accept 1 value.
// 	// By default channels are unbuffered, meaning that they will only accept
// 	// sends (chan <-) if there is a corresponding receive (<- chan) ready to
// 	// receive the sent value. Buffered channels accept a limited number of
// 	// values without a corresponding receiver for those values.
// 	out := make(chan string, 1)

// 	go func() {
// 		defer f.Close()
// 		defer close(out)

// 		var sb strings.Builder

// 		for {
// 			data := make([]byte, 8)
// 			n, err := f.Read(data)

// 			if err != nil {
// 				break
// 			}

// 			for i := range n {
// 				if data[i] == '\n' {
// 					out <- sb.String()
// 					sb.Reset()
// 				} else {
// 					sb.WriteByte(data[i])
// 				}
// 			}
// 		}

// 		if sb.Len() > 0 {
// 			out <- sb.String()
// 		}
// 	}()

// 	return out
// }

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			fmt.Printf("connection accepted\n")

			r, err := request.RequestFromReader(c)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Request line:\n")
			fmt.Printf("- Method: %s\n", r.RequestLine.Method)
			fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
			fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)

			fmt.Printf("Headers:\n")
			for key, value := range r.Headers {
				fmt.Printf(" - %s: %s\n", key, value)
			}

			fmt.Printf("Body:\n")
			fmt.Printf("%s\n", string(r.Body))

			c.Close()
			fmt.Printf("connection closed\n")
		}(conn)

		// go func(c net.Conn) {
		// 	fmt.Printf("connection accepted\n")

		// 	lines := getLinesFromReader(c)
		// 	for line := range lines {
		// 		fmt.Printf("%s\n", line)
		// 	}

		// 	c.Close()
		// 	fmt.Printf("connection closed\n")
		// }(conn)
		// fmt.Printf("connection accepted\n")

		// lines := getLinesFromReader(conn)
		// for line := range lines {
		// 	fmt.Printf("%s\n", line)
		// }

		// conn.Close()
		// fmt.Printf("connection closed\n")
	}
}
