package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

type Headers struct {
	X_REQUESTID  string `json:"x_request_id"`
	X_FILENAME   string `json:"x_filename"`
	X_PATH       string `json:"x_path"`
	X_CHUNK_SIZE int64  `json:"x_chunk"`
	X_NBYTES     int64  `json:"x_nbytes"`
	X_PROTOCOL   string `json:"x_protocol"`
}

func Server() {
	addr := ":9988"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server listening on", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

type ConnectionPool struct {
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var h Headers
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&h); err != nil {
		fmt.Println("Error decoding headers:", err)
		return
	}

	fmt.Printf("Received headers: %+v\n", h)

	// Ensure the directory exists
	if err := os.MkdirAll(h.X_PATH, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	// Receive the file content in chunks
	totalBytes := int64(0)
	// chunk := 1
	// Create the file
	filePath := filepath.Join(h.X_PATH, h.X_FILENAME)
	file, err := os.OpenFile(filePath, os.O_SYNC|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	for {
		written, err := io.CopyN(file, conn, h.X_NBYTES)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error writing to file:", err)
			return
		}
		if written > h.X_NBYTES {
			break
		}
		fmt.Printf("\r\rReceived %v %%", (totalBytes*100)/h.X_NBYTES)
	}
}

func Client(filePath string) {
	conn, err := net.Dial("tcp", ":9988")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		panic(err)
	}
	size := fileInfo.Size()

	headers := Headers{
		X_REQUESTID:  "1111-22222-33333-444444-555555",
		X_FILENAME:   filePath,
		X_PATH:       "./received_files",
		X_CHUNK_SIZE: 100,
		X_NBYTES:     size,
		X_PROTOCOL:   "tcp",
	}

	data, err := json.Marshal(&headers)
	if err != nil {
		panic(err)
	}
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}

	file, err := os.OpenFile(filePath, os.O_SYNC|os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	totalBytes := int64(0)
	for {
		sent, err := io.CopyN(conn, file, size)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error sending file:", err)
			return
		}
		totalBytes += sent
		if totalBytes >= size {
			break
		}
	}

	fmt.Println("File sent successfully:", filePath)
}

func main() {
	var op string
	var fileName string
	flag.StringVar(&op, "name", "server", "Specify 'server' or 'client'")
	flag.StringVar(&fileName, "file", "main.bin", "Specify 'server' or 'client'")
	flag.Parse()

	if op == "server" {
		Server()
	} else {
		Client(fileName)
	}
}
