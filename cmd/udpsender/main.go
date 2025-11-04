package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	raddr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// The built-in bufio package implements buffered IO operations.
	//
	// Since system calls are costly, the performance of IO operations
	// is greatly improved when we accumulate data into a buffer where
	// reading or writing it. This reduces the number of system calls
	// needed.
	//
	// Without buffering: If your program reads 1 byte at a time, each
	// read triggers a system call - switching from user mode to kernel
	// mode, which is expensive. Reading 1000 bytes means 1000 system calls.
	//
	// With buffering: The buffer performs one larger system call to read,
	// say, 4KB of data into memory. Your program then reads from this
	// in-memory buffer byte-by-byte without any system calls. Only when
	// the buffer is empty does another system call happen to refill it.
	// Reading 1000 bytes might only need 1 system call instead of 1000.
	//
	// The same applies to writes - you accumulate data in the buffer and
	// flush it to the OS in one batch rather than making a system call for
	// each small write.
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println(">")

		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Fatal(err)
		}
	}
}
