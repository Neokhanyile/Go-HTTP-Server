package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening on port 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		log.Print("error reading request: ", err)
		return
	}

	requestStr := string(buf[:n])
	fmt.Printf("Received:\n%s\n", requestStr)

	lines := strings.Split(requestStr, "\r\n")
	if len(lines) == 0 {
		log.Print("empty request")
		return
	}

	requestLine := lines[0]
	parts := strings.SplitN(requestLine, " ", 3)
	if len(parts) < 3 {
		log.Print("malformed request line")
		return
	}

	method := parts[0]
	rawPath := parts[1]
	version := strings.TrimSpace(parts[2])

	path := strings.SplitN(rawPath, "?", 2)[0]

	fmt.Printf("Method: %s\n", method)
	fmt.Printf("Path: %s\n", path)
	fmt.Printf("Version: %s\n", version)

	headers := make(map[string]string)
	for _, line := range lines[1:] {
		if line == "" {
			break
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])
		headers[key] = value
	}

	fmt.Println("Headers:")
	for key, value := range headers {
		fmt.Printf("  %s: %s\n", key, value)
	}

	bodyStart := strings.Index(requestStr, "\r\n\r\n")
	if bodyStart == -1 {
		log.Print("malformed request: no header/body boundary found")
		return
	}
	bodyStart += 4

	var body []byte

	contentLengthStr, ok := headers["content-length"]
	if ok {
		cl, err := strconv.Atoi(contentLengthStr)
		if err == nil && cl > 0 {
			alreadyRead := n - bodyStart

			body = append(body, buf[bodyStart:n]...)

			if cl > alreadyRead {
				extra := make([]byte, cl-alreadyRead)
				_, err := io.ReadFull(conn, extra)
				if err != nil {
					log.Print("error reading body: ", err)
					return
				}
				body = append(body, extra...)
			}

			fmt.Printf("Body: %s\n", string(body))
		}
	}

	var responseBody string
	var statusCode int
	var statusText string

	switch path {
	case "/":
		responseBody = "Home page"
		statusCode = 200
		statusText = "OK"
	case "/hello":
		responseBody = "Hello, world!"
		statusCode = 200
		statusText = "OK"
	case "/echo":
		responseBody = string(body)
		statusCode = 200
		statusText = "OK"
	default:
		responseBody = "404 page not found"
		statusCode = 404
		statusText = "Not Found"
	}

	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		statusCode, statusText, len(responseBody), responseBody,
	)

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Print("error writing response: ", err)
	}
}
