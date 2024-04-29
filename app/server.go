package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	RESP_ARRAY       rune = '*'
	RESP_BULK_STRING rune = '$'
)
const SEPARATOR = "\r\n"

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379", err.Error())
		os.Exit(1)
	}
	fmt.Println("Redis server started" + listener.Addr().String())
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
		}
		go handleClientConnection(conn)
	}
}

func handleClientConnection(conn net.Conn) {
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Connection closed")
				break
			}
			fmt.Println("Error reading from connection: ", err.Error())
		}
		data := string(buf[:n])
		rsp := deserializeResp(data)
		args := rsp.Data.([]*RespData)
		cmd := args[0].Data.(string)
		argsLength := len(args)
		switch strings.ToLower(cmd) {
		case "ping":
			conn.Write([]byte("+PONG\r\n"))
		case "echo":
			if argsLength < 2 {
				conn.Write([]byte("+ \r\n"))
				continue
			}

			conn.Write([]byte("+" + args[1].Data.(string) + "\r\n"))
		default:
			conn.Write([]byte("-ERR unknown command '" + cmd + "'\r\n"))
		}

	}
}

func deserializeResp(str string) *RespData {
	result := strings.Split(str, SEPARATOR)
	resp := parseRsp(result)
	return resp
}

func parseRsp(data []string) *RespData {
	if len(data) < 1 || len(data[0]) < 1 {
		return nil
	}
	resp_type := rune(data[0][0])
	switch resp_type {
	case RESP_BULK_STRING:
		return parseBulkString(data)
	case RESP_ARRAY:
		if len(data[0]) < 2 {
			return nil
		}
		return parseRspArray(data)
	}
	return nil
}

func parseRspArray(arr []string) *RespData {
	length, err := strconv.Atoi(arr[0][1:])
	if err != nil {
		return nil
	}
	messages := make([]*RespData, 0, length)
	for i := 1; i <= length*2; i = i + 2 {
		messages = append(messages, parseRsp(arr[i:]))
	}
	return &RespData{Type: RESP_ARRAY, Data: messages}
}

func parseBulkString(data []string) *RespData {
	if len(data) < 2 {
		return nil
	}
	length, err := strconv.Atoi(data[0][1:])
	if err != nil {
		return nil
	}
	if len(data[1]) != length {
		return nil
	}

	return &RespData{Type: RESP_BULK_STRING, Data: data[1]}
}

func parseRespDataToString(resp *RespData) string {
	return resp.Data.(string)
}

type RespData struct {
	Type rune
	Data any
}
