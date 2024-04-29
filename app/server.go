package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
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
	handleClientConnection(conn)
}

func handleClientConnection(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
		}
		fmt.Println("Received data: ", buf[:n])
		conn.Write([]byte("+PONG\r\n"))
		fmt.Println("Received connection")
	}
}
