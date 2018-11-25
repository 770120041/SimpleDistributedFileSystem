package transfer

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"../file"
)

//Commands
const (
	SEND = iota
	DOWNLOAD
)

//StartTransferServer starts the tcp server for upload/download of data from this node.
func StartTransferServer(port string, localStorage *file.FileStorage) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Could not start server!")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("Could not accept connection !")
		}
		go connectionHandler(conn, port, localStorage)
	}
}

func connectionHandler(conn net.Conn, port string, localStorage *file.FileStorage) {
	command := make([]byte, 1)
	n, err := conn.Read(command)
	if n != 1 || err != nil {
		log.Fatal("Could not read command bytes", err)
	}
	clientCommand := command[0]
	switch clientCommand {
	case SEND:
		receiveDataHandler(conn, createFile, writeChunk, port, localStorage)
	case DOWNLOAD:
		uploadDataHandler(conn, openFile, port)
	default:
		log.Println("Unknown command")
		conn.Close()
	}

}

func uploadDataHandler(conn net.Conn, openFileHandler func(string, string) *os.File, port string) {
	startTime := time.Now()
	defer conn.Close()

	//Read the first 1024 bytes to find the filenamme to send back
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n != 1024 {
		log.Println("Error reading:", err.Error(), n)
	}
	sdfsFile := strings.TrimSpace(string(bytes.Trim(buf[0:1015], "\x00")))

	fileToSend := openFileHandler(sdfsFile, port)
	defer fileToSend.Close()

	dataBuffer := make([]byte, 65535)
	sentBytes := 0
	for {
		n, err := fileToSend.Read(dataBuffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Upload complete. EOF reached")
				break
			}
			log.Fatal("Error reading data from file", err)
		}
		n, err = conn.Write(dataBuffer[:n])
		if err != nil {
			log.Fatal("Could not send data!")
		}
		sentBytes += n
	}

	log.Println("Uploaded", sentBytes/1000000, "MB of data.", "Transfer time", time.Now().Sub(startTime))
}

func receiveDataHandler(conn net.Conn, createFileHandler func(string) *os.File,
	writeChunkHandler func(*os.File, []byte) int, port string, localStorage *file.FileStorage) {
	startTime := time.Now()
	defer conn.Close()

	//Read the first 1024 bytes to establish filename and filesize
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if n != 1024 || err != nil {
		log.Println("Error reading:", err.Error(), n)
	}
	filename := strings.TrimSpace(string(bytes.Trim(buf[0:1015], "\x00")))
	filesize, err := binary.ReadVarint(bytes.NewBuffer(buf[1016:1024]))
	if err != nil {
		log.Println("filesize conversion failed", err)
	}

	filenames := strings.Split(filename, ">>")
	originName, storeName := filenames[0], filenames[1]
	newFile := createFileHandler("/tmp/sdfs/sdfs-" + port + "/" + storeName) //always store files in sdfs folder
	defer newFile.Close()

	//Read incoming data in chunks of 65535 bytes and store them to file

	buf = make([]byte, 65535)
	accumaltedBytesRead := int64(0)

	for accumaltedBytesRead != filesize {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("Transfer complete. EOF reached")
				break
			}
			log.Fatal("Error reading incoming data", err)
		}
		writeChunkHandler(newFile, buf[:n])
		accumaltedBytesRead += int64(n)
	}

	log.Println("Read", accumaltedBytesRead/1000000, "MB of data", "Transfer time", time.Now().Sub(startTime))
	localStorage.AddLocalFile(originName, storeName)
}
