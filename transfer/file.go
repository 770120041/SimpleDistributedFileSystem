package transfer

import (
	"log"
	"os"
)

func createFile(filename string) *os.File {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Something went wrong with file", err)

	}
	return file
}

func openFile(sdfsname, port string) *os.File {
	file, err := os.Open("/tmp/sdfs/sdfs-" + port + "/" + sdfsname) //Always read files from sdfs folder.
	if err != nil {
		log.Fatal("Could not read file", err)
	}
	return file
}

func writeChunk(file *os.File, chunk []byte) int {
	n, err := file.Write(chunk)
	if err != nil {
		log.Fatal("Could not write file", err)
	}
	return n
}

func fileNameBuffer(filename string) []byte {
	//first 1016 bytes will contain the filename, the last 8 bytes will contain the filesize
	fileInfoBuffer := make([]byte, 1024)
	fileNamebuffer := []byte(filename)
	copy(fileInfoBuffer, fileNamebuffer)
	return fileInfoBuffer
}
