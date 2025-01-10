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
	chunk := 1
	for totalBytes < h.X_NBYTES {

		bytesToWrite := (h.X_NBYTES - totalBytes)
		if bytesToWrite > h.X_CHUNK_SIZE*1024*1024 {
			bytesToWrite = h.X_CHUNK_SIZE * 1024 * 1024

		}
		// Create the file
		filePath := filepath.Join(h.X_PATH, fmt.Sprintf("%s.%s", h.X_FILENAME, chunk))
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()
		chunk++
		written, err := io.CopyN(file, conn, bytesToWrite)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error writing to file:", err)
			return
		}

		totalBytes += written
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

	// Send headers as JSON
	data, err := json.Marshal(&headers)
	if err != nil {
		panic(err)
	}
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}

	// Send the file content in chunks
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	totalBytes := int64(0)
	for {
		bytesToSend := size - totalBytes
		if bytesToSend > headers.X_CHUNK_SIZE*1024*1024 { // Limit chunks to 4MB
			bytesToSend = headers.X_CHUNK_SIZE * 1024 * 1024
		}

		sent, err := io.CopyN(conn, file, bytesToSend)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error sending file:", err)
			return
		}

		totalBytes += sent

		fmt.Printf("\r\rSent %v %%", (totalBytes*100)/size)
		//fmt.Printf("Sent %d/%d bytes\n", totalBytes, size)

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
