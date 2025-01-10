// package main

// import (
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"io"
// 	"net"
// 	"os"
// )

// type Headers struct {
// 	X_REQUESTID string
// 	X_FILENAME  string
// 	X_PATH      string
// 	X_CHUNK     string
// 	X_NBYTES    int64
// 	X_PROTOCOL  string
// }

// func NewHeaders() *Headers {
// 	return &Headers{
// 		X_REQUESTID: "x_request_id",
// 		X_FILENAME:  "x_filename",
// 		X_PATH:      "x_path",
// 		X_CHUNK:     "x_chunk",
// 		X_NBYTES:    int64(100),
// 		X_PROTOCOL:  "x_protocol",
// 	}
// }

// func Server() {
// 	addr := "127.0.0.1:9988"
// 	listner, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer listner.Close()
// 	for {
// 		conn, err := listner.Accept()
// 		if err != nil {
// 			panic(err)
// 		}

// 		var h Headers
// 		decoder := json.NewDecoder(conn)
// 		decoder.Decode(&h)
// 		fmt.Println(h)
// 		go HandelData(conn, &h)
// 	}
// }

// func HandelData(conn net.Conn, h *Headers) {
// 	defer conn.Close()
// 	file, err := os.Create(fmt.Sprintf("%s/%s", h.X_PATH, h.X_FILENAME))
// 	if err != nil {
// 		panic(err)
// 	}
// 	for {
// 		fmt.Println("hello world ....")
// 		_, err := io.CopyN(file, conn, h.X_NBYTES)
// 		if err != nil {
// 			break
// 		}
// 	}
// 	fmt.Println("headers : ", h)
// }

// func Client() {

// 	conn, err := net.Dial("tcp", "127.0.0.1:9988")
// 	if err != nil {
// 		panic(err)
// 	}

// 	fileInfo, err := os.Stat("main.bin")
// 	if err != nil {
// 		panic(err)
// 	}
// 	size := fileInfo.Size()

// 	data, err := json.Marshal(&Headers{
// 		X_REQUESTID: "1111-22222-33333-444444-555555",
// 		X_FILENAME:  "main.bin",
// 		X_PATH:      "./",
// 		X_CHUNK:     "1",
// 		X_NBYTES:    int64(size),
// 		X_PROTOCOL:  "tcp",
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	conn.Write(data)
// 	file, err := os.Open("send.bin")
// 	if err != nil {
// 		panic(err)
// 	}
// 	for {
// 		nsize, err := io.CopyN(file, conn, size)
// 		if err != nil {
// 			break
// 		}
// 		fmt.Println(nsize)
// 	}
// }

// func main() {
// 	var op string
// 	flag.StringVar(&op, "name", "server", "")
// 	flag.Parse()

// 	if op == "server" {
// 		Server()
// 		return
// 	}
// 	Client()
// }

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
	X_REQUESTID string `json:"x_request_id"`
	X_FILENAME  string `json:"x_filename"`
	X_PATH      string `json:"x_path"`
	X_CHUNK     string `json:"x_chunk"`
	X_NBYTES    int64  `json:"x_nbytes"`
	X_PROTOCOL  string `json:"x_protocol"`
}

func Server() {
	addr := "127.0.0.1:9988"
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

	// Create the file
	filePath := filepath.Join(h.X_PATH, h.X_FILENAME)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Receive the file content in chunks
	totalBytes := int64(0)
	for totalBytes < h.X_NBYTES {
		bytesToWrite := (h.X_NBYTES - totalBytes)
		if bytesToWrite > 4*1024*1024 {
			bytesToWrite = 4 * 1024 * 1024
		}

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

	fmt.Println("File received successfully:", filePath)
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
		X_REQUESTID: "1111-22222-33333-444444-555555",
		X_FILENAME:  filePath,
		X_PATH:      "./received_files",
		X_CHUNK:     "1",
		X_NBYTES:    size,
		X_PROTOCOL:  "tcp",
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
		if bytesToSend > 4*1024*1024 { // Limit chunks to 4MB
			bytesToSend = 4 * 1024 * 1024
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
