package transfer

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"time"

	"../leader"
)

//SendData sends a localfile to a nodes sdfs storage
func SendData(filepath, sdfsFileName, dstFileName, dstID string) {
	startTime := time.Now()
	conn, err := net.Dial("tcp", leader.IDtoIP(dstID))
	if err != nil {
		log.Fatal("could not connect to server", err)
	}

	// send a byte with the command
	conn.Write([]byte{SEND})

	//open file
	file, err := os.Open(filepath)
	defer file.Close()
	if err != nil {
		log.Fatal("Could not open the file", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal("Could not get file info", err)
	}

	fileInfoBuffer := fileNameBuffer(sdfsFileName + ">>" + dstFileName)

	fileSizeBuf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(fileSizeBuf, fileInfo.Size())
	copy(fileInfoBuffer[1016:1024], fileSizeBuf)

	if len(fileInfoBuffer) != 1024 {
		log.Fatal("File info buffer not 1024 bytes")
	}
	_, err = conn.Write(fileInfoBuffer)
	if err != nil {
		log.Fatal("Could not write fileinformation", err)
	}

	//send the file in chunks of 65535 bytes
	accumulatedBytes := 0
	fileBuffer := make([]byte, 65535)
	for {

		n, err := file.Read(fileBuffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Something went wrong!", err)
		}
		transferChunk(fileBuffer[:n], conn)
		accumulatedBytes += n
	}

	log.Println("Sent", accumulatedBytes/1000000, "MB. Transfer time", time.Now().Sub(startTime))
	conn.Close()
}

func transferChunk(chunk []byte, conn net.Conn) {
	_, err := conn.Write(chunk)
	if err != nil {
		log.Fatal("Something bad happened when writing data")
	}
}
