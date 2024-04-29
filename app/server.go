package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.

	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379", err.Error())
		os.Exit(1)
	}

	fmt.Println("Redis server started on port 6379")
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection:", err)
	}

	conn.Write([]byte("+PONG\r\n"))
}
