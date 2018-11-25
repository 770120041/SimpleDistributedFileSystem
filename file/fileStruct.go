package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
)

// the db have the dbfile slice
// each dbfile knows the sdfsFile name and different versions
// each version will know where it is stored and
type FileDB []dbFile
type dbFile struct {
	sdfsName string
	versions []FileDataEntry
}

type FileDataEntry struct {
	storedName string //hased of sdfsfilename | versionnumber
	Info       []storedInfo
}

type storedInfo struct {
	storedNode  string
	storedState int
	//0 not ACK
	//1 ACK
	//-1 not ACK, delete later
}

//local storage
type FileStorage []LocalEntry

type LocalEntry struct {
	OriginName string `json:"OriginName:"`
	StoredName string `json:"StoredName"`
}

func head(slice []FileDataEntry) (FileDataEntry, error) {
	if len(slice) == 0 {
		return FileDataEntry{}, errors.New("Can not take head of empty slice")
	}
	return slice[0], nil
}

func (filedb *FileDB) GetNodesThatStore(sdfs string) []string {

	fileIndex := filedb.findDBFileByName(sdfs)
	if fileIndex == -1 {
		log.Println("No one is storing", sdfs)
		return []string{}
	}
	file := (*filedb)[fileIndex]
	nodes := []string{}
	fileVersion, err := head(file.versions)
	if err != nil {
		log.Println("No one is storing", sdfs)
		return []string{}
	}
	for _, v := range fileVersion.Info {
		nodes = append(nodes, v.storedNode)
	}
	return nodes
}

//get a file by name, return index
func (filedb *FileDB) findDBFileByName(sdfsFileName string) int {
	for i, entry := range *filedb {
		if entry.sdfsName == sdfsFileName {
			return i
		}
	}
	return -1
}

func (filedb *FileDB) findDBFileVersion(fileIndex int, dstName string) int {
	if fileIndex == -1 {
		return -1
	}
	if fileIndex >= len(*filedb) {
		return -1
	}
	for i, version := range (*filedb)[fileIndex].versions {
		if version.storedName == dstName {
			return i
		}
	}
	return -1
}

//get info number by dst Node
//sdfsFilename,version,dstNode
func (filedb *FileDB) findDBInfoNumber(fileIdx, storedVersionIdx int, dstNode string) int {
	if fileIdx == -1 || storedVersionIdx == -1 {
		return -1
	}
	if fileIdx >= len(*filedb) {
		return -1
	}
	if storedVersionIdx >= len((*filedb)[fileIdx].versions) {
		return -1
	}
	for i, storedInfo := range (*filedb)[fileIdx].versions[storedVersionIdx].Info {
		if storedInfo.storedNode == dstNode {
			return i
		}
	}

	return -1
}

func newDBVersion(dstFileName, newStoredNode string) *FileDataEntry {
	var entry FileDataEntry
	entry.storedName = dstFileName
	entry.Info = append(entry.Info, storedInfo{newStoredNode, 0})
	return &entry
}
func newDBFile(sdfsFilename string, dstFileName string, newStoredNode string) *dbFile {
	var file dbFile
	file.sdfsName = sdfsFilename
	file.versions = append(file.versions, *newDBVersion(dstFileName, newStoredNode))
	return &file
}
func (filedb *FileDB) addStoredNode(sdfsFileName, dstFileName, storedDst string) {
	fileIndex := filedb.findDBFileByName(sdfsFileName)

	//a new sdfs file name
	if fileIndex == -1 {
		log.Println("database stored a new sdfsfile named:", sdfsFileName)
		*filedb = append(*filedb, *newDBFile(sdfsFileName, dstFileName, storedDst))
	} else {
		//find if an older or newer version of the same sdfsFile
		storedVersion := filedb.findDBFileVersion(fileIndex, dstFileName)

		//older versions
		if storedVersion != -1 {
			log.Println("database now is storing:", sdfsFileName, " of version:", storedVersion+1)
			(*filedb)[fileIndex].versions[storedVersion].Info = append((*filedb)[fileIndex].versions[storedVersion].Info, storedInfo{storedDst, 0})
		} else {
			log.Println("database now is creating a new version for:", sdfsFileName, " of versionNumber:", len((*filedb)[fileIndex].versions))
			(*filedb)[fileIndex].versions = append((*filedb)[fileIndex].versions, *newDBVersion(dstFileName, storedDst))
		}

	}

}

func (filedb *FileDB) ShowDB() {
	db := *filedb
	for _, file := range db {
		log.Println("the sdfsName is :", file.sdfsName)
		for _, version := range file.versions {
			log.Println("the storedName is :", version.storedName)
			for _, entry := range version.Info {
				log.Println("\t\t\tNodeID:", entry.storedNode+",", "NodeState:", entry.storedState, " ")
			}
		}
	}
}

/**
	Remove elements in DB
**/

func (filedb *FileDB) deleteDBStoredNode(fileIdx, storedVersionIdx, InfoNumber int) {
	(*filedb)[fileIdx].versions[storedVersionIdx].Info = append((*filedb)[fileIdx].versions[storedVersionIdx].Info[:InfoNumber], (*filedb)[fileIdx].versions[storedVersionIdx].Info[InfoNumber+1:]...)
}

/**
	tools for local file storage
**/
func (storage *FileStorage) IsExist(storedName string) bool {
	for _, entry := range *storage {
		if entry.StoredName == storedName {
			return true
		}
	}
	return false
}
func (storage *FileStorage) AddLocalFile(originName, storedName string) {
	log.Println("Save", originName, "locally")
	localEntry := LocalEntry{originName, storedName}
	*storage = append(*storage, localEntry)
}
func (storage *FileStorage) RemoveFileStorage(fileName string) {
	fileStorage := *storage
	for i, localEntry := range fileStorage {
		if localEntry.StoredName == fileName {
			log.Println("remove slice at pos:", i)
			*storage = append((*storage)[:i], (*storage)[i+1:]...)
			break
		}
	}
}

func (storage *FileStorage) ShowLocalFile() {
	fileStorage := *storage
	for _, localEntry := range fileStorage {
		log.Println("Stored file originName:", localEntry.OriginName, ", and stored name:", localEntry.StoredName)
	}
}

func JsonLocalStorage(localStorage FileStorage) string {
	storageMar, err := json.Marshal(localStorage)
	if err != nil {
		log.Fatal(err, " malfored local storage")
	}
	return string(storageMar)
}

func UnjsonLocalStorage(input []byte) (localStorage FileStorage) {
	input = bytes.Trim(input, "\x00")
	err := json.Unmarshal(input, &localStorage)
	if err != nil {
		log.Fatal(err, " Have problem Unmarshall the message")
		return localStorage
	}
	return localStorage
}
