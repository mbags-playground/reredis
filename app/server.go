package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	RESP_ARRAY       rune = '*'
	RESP_BULK_STRING rune = '$'
)
const SEPARATOR = "\r\n"
const (
	SYNTAX_ERROR string = "syntax error"
)

type RespData struct {
	Type rune
	Data any
}
type MemoryData struct {
	expires time.Time
	data    RespData
}

var memory = make(map[string]MemoryData)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379", err.Error())
		os.Exit(1)
	}

	fmt.Println("Server started, waiting for tcp connection" + listener.Addr().String())
	defer listener.Close()
	startConnection(listener)
}

func startConnection(listener net.Listener) {
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
		case "set":
			if argsLength < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
				continue
			}
			key := args[1].Data.(string)
			value := args[2].Data.(string)
			keepTTL := false

			hasNx := false
			hasXx := false
			var expires time.Time

			if argsLength == 3 {
				memory[key] = MemoryData{expires: expires, data: RespData{Type: RESP_BULK_STRING, Data: value}}
				fmt.Println(expires.IsZero())
			}
			if argsLength == 4 {
				option := args[3].Data.(string)
				switch option {

				case "KEEPTTL":
					keepTTL = true
				case "NX":
					hasNx = true
				case "XX":
					hasXx = true
				default:
					conn.Write(toError(SYNTAX_ERROR))
				}
			}
			if argsLength == 5 || argsLength == 6 {
				option := strings.ToUpper(args[3].Data.(string))
				t, ok := args[4].Data.(string)
				if !ok {
					conn.Write(toError(SYNTAX_ERROR))
				}
				str, err := strconv.Atoi(t)
				if err != nil {
					conn.Write(toError(SYNTAX_ERROR))
				}
				if !ok {
					conn.Write(toError("Invalid format"))
				}
				switch option {
				case "EX":
					expires = time.Now().Add(time.Duration(str) * time.Second)
				case "PX":
					expires = time.Now().Add(time.Duration(str) * time.Millisecond)
				default:
					conn.Write(toError(SYNTAX_ERROR))

				}
				if argsLength == 6 {
					option = strings.ToUpper(args[5].Data.(string))
					switch option {
					case "NX":
						hasNx = true
					case "XX":
						hasXx = true
					default:
						conn.Write(toError(SYNTAX_ERROR))
					}
				}

			}

			memory[key] = MemoryData{expires: expires, data: RespData{Type: RESP_BULK_STRING, Data: value}}
			fmt.Println(keepTTL, hasNx, hasXx)
			conn.Write([]byte("+OK\r\n"))
		case "get":
			var expires time.Time
			if argsLength < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := args[1].Data.(string)
			memoryValue, ok := memory[key]
			if !ok || memoryValue.data.Data == nil {
				conn.Write([]byte("$-1\r\n"))
				continue
			}
			expires = memoryValue.expires
			if !expires.IsZero() && time.Now().After(expires) {
				delete(memory, key)
				conn.Write([]byte("$-1\r\n"))
				continue
			}
			value, ok := memoryValue.data.Data.(string)
			if !ok {
				conn.Write([]byte("$-1\r\n"))
				continue
			}
			conn.Write([]byte("+" + value + "\r\n"))

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

func toError(message string) []byte {
	return []byte("-ERR" + message + "\r\n")
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
