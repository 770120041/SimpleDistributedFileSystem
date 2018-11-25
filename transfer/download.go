package transfer

import (
	"io"
	"log"
	"net"
	"time"

	"../leader"
)

//DownloadData connects to a node and downloads a file froms its sdfs storage and saves it as localfile
func DownloadData(localfile, dstFile, dstID string) {
	startTime := time.Now()
	conn, err := net.Dial("tcp", leader.IDtoIP(dstID))
	if err != nil {
		log.Fatal("Could not connect to the server", err)
	}

	//send a byte to set the server in correct mode..
	conn.Write([]byte{DOWNLOAD})

	//send a request for the filename to the serve	r
	fileRequestBuf := fileNameBuffer(dstFile)
	conn.Write(fileRequestBuf)

	//Create a file to write the downloaded data to
	newFile := createFile(localfile)
	defer newFile.Close()

	buf := make([]byte, 65535)
	accumaltedBytesRead := int64(0)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("Download complete. EOF reached.")
				break
			}
		}
		writeChunk(newFile, buf[:n])
		accumaltedBytesRead += int64(n)
	}
	log.Println("Downloaded", accumaltedBytesRead/1000000, "MB of data. Transfer time:", time.Now().Sub(startTime))
	conn.Close()
}
