package file

import (
	"log"
	"os"
)

var folderPath = "/tmp/sdfs/sdfs-"

//CreatesdfsFolder run this in main.
func CreateSdfsFolder(port string) {
	folderPath = "/tmp/sdfs/sdfs-" + port + "/"
	os.RemoveAll(folderPath)
	os.MkdirAll(folderPath, 0777)
}

//DeleteFile will delete a file form the sdfs folder.
func DeleteFile(sdfsfilename string) {
	log.Println("Delete file at", folderPath+sdfsfilename)
	os.Remove(folderPath + sdfsfilename)
}
