package file

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"../com"
)

/**
	tools
**/
func hashedFileName(filename string) string {
	md5HashInBytes := md5.Sum([]byte(filename))
	md5HashInString := hex.EncodeToString(md5HashInBytes[:])
	return md5HashInString
}

/**
	When transmiting message about file, all the same format
	srcfileName,dstFilename,srcID,dstID,
	the deliminater is >>
**/

/**
	Store Section
**/
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sendStoreRequestMsg(masterID, filePath, sdfsFileNamestring, dstFileName, srcID, dstID string) {
	dstMsg := com.Msg{com.SENDREQUEST, masterID, makeStoreMsg(filePath, sdfsFileNamestring, dstFileName, srcID, dstID)}
	com.SendMsg(dstMsg, srcID, 0)
}
func (filedb *FileDB) StoreSingleFile(masterID string, filePath, sdfsFileName, dstFileName, srcID, dstID string) {

	filedb.addStoredNode(sdfsFileName, dstFileName, dstID)
	sendStoreRequestMsg(masterID, filePath, sdfsFileName, dstFileName, srcID, dstID)
}

func (filedb *FileDB) StoreNewFile(masterID string, filePath, sdfsFileNamestring, srcID string, membership []string) {

	fileIndex := filedb.findDBFileByName(sdfsFileNamestring)
	curVersions := 0
	if fileIndex != -1 {
		curVersions = len((*filedb)[fileIndex].versions)
		log.Println("newVersion of Same file, versionNumebr is:", curVersions)
	}
	dstFileName := hashedFileName(sdfsFileNamestring) + "|" + strconv.Itoa(curVersions)
	targetNode := membership[rand.Intn(len(membership))]
	replicaNumber := 4
	if len(membership) < 4 {
		replicaNumber = len(membership)
	}
	//send storage request to one random node and 3 subsequent nodes
	for i := 0; i < replicaNumber; i++ {
		// not stored replica files
		for filedb.findDBInfoNumber(fileIndex, curVersions, targetNode) != -1 {
			targetNode = findNextHop(targetNode, membership)
			fmt.Println("once")
		}
		filedb.StoreSingleFile(masterID, filePath, sdfsFileNamestring, dstFileName, srcID, targetNode)
		targetNode = findNextHop(targetNode, membership)
		fmt.Println("nextNode is :", targetNode)

	}

}
func makeStoreMsg(filePath, srcFileName, dstFileName, srcID, dstID string) string {
	return filePath + ">>" + srcFileName + ">>" + dstFileName + ">>" + srcID + ">>" + dstID
}

func ParseStoreMsg(msgContent string) (string, string, string, string, string) {
	contents := strings.Split(msgContent, ">>")
	pathName := contents[0]
	sdfsFileName := contents[1]
	dstFileName := contents[2]
	srcID := contents[3]
	dstID := contents[4]
	return pathName, sdfsFileName, dstFileName, srcID, dstID
}

/**
	GET section
**/
func (filedb *FileDB) GetNodeForFile(fileIdx int, targetVersionNumber int) (string, string) {
	versions := (*filedb)[fileIdx].versions
	for _, version := range versions {
		sdfsName := version.storedName
		names := strings.Split(sdfsName, "|")
		versionNumber, _ := strconv.Atoi(names[1])
		if versionNumber == targetVersionNumber {
			return sdfsName, version.Info[0].storedNode
		}
	}
	log.Println("didn't get desired version")
	return "", ""
}
func sendGetRespond(requestNode, localName, storedName, storedNode, masterID string) {
	dstMsg := com.Msg{com.GETRESPOND, masterID, storedNode + ">>" + storedName + ">>" + localName}
	com.SendMsg(dstMsg, requestNode, 0)
}

func sendFailedGetRespond(requestNode, masterID string) {
	dstMsg := com.Msg{com.GETRESPOND, masterID, "missing"}
	com.SendMsg(dstMsg, requestNode, 0)
}

func (filedb *FileDB) GetDBFileRequest(masterID, sdfsname, localName, versionString, requestNode string) {

	fileIdx := filedb.findDBFileByName(sdfsname)
	if fileIdx == -1 {
		log.Println("no such file", sdfsname, "exist in DB, cannot get this file")
		sendFailedGetRespond(requestNode, masterID)
		return
	}

	desiredNumber, _ := strconv.Atoi(versionString)

	if desiredNumber < 0 {
		desiredNumber = len((*filedb)[fileIdx].versions) - 1
		storedName, storedNode := filedb.GetNodeForFile(fileIdx, desiredNumber)
		sendGetRespond(requestNode, localName, storedName, storedNode, masterID)
	} else {
		maxVersionNumber := len((*filedb)[fileIdx].versions)
		for i := maxVersionNumber - 1; i >= maxVersionNumber-desiredNumber; i-- {
			//TODO handle here
			storedName, storedNode := filedb.GetNodeForFile(fileIdx, i)
			sendGetRespond(requestNode, localName+"|"+strconv.Itoa(i), storedName, storedNode, masterID)
			fmt.Println("asking version :", i, "from ", storedNode)
		}
	}

}

/**
	Check Status Secion
**/
// func sendStorePingMsg(srcFileName, dstFileName, dstID, masterID string) {
// 	dstMsg := com.Msg{com.STOREPING, masterID, srcFileName + ">>" + dstFileName}
// 	com.SendMsg(dstMsg, dstID, 0)
// }

// func MakeStorePingRespondMsg(sdfsName, storedName, senderID string, state int) string {
// 	return sdfsName + ">>" + storedName + ">>" + strconv.Itoa(state)
// }

func (filedb *FileDB) DBStoreUpdate(masterID string, membership []string, deletedNode string) {
	db := *filedb

	if deletedNode != "REGULAR" {
		log.Println("node:", deletedNode, " was deleted, master need to update database")
		filedb.DBDeleteFileByNode(deletedNode)
	}
	for i, file := range db {
		for j, version := range file.versions {
			var oneStoredNode = ""
			for _, entry := range version.Info {
				oneStoredNode = entry.storedNode
				break
			}
			if oneStoredNode == "" {
				log.Println("no replica for :", file.sdfsName, ", DB crashed before replication")
			}
			//if less than 4, replicate by it self
			if len(version.Info) < 4 {
				fileIndex := i
				curVersionNumber := j
				existReplicaNumber := len((*filedb)[fileIndex].versions[curVersionNumber].Info)
				replicaNumber := 4 - existReplicaNumber
				if len(membership)-existReplicaNumber < replicaNumber {
					replicaNumber = len(membership) - existReplicaNumber
				}
				log.Println("exist replica numbers:", existReplicaNumber)
				targetNode := membership[rand.Intn(len(membership))]

				if oneStoredNode == "" {
					log.Fatal("nobody stores file", file.sdfsName, " , it is lost")
				}

				for i := 0; i < replicaNumber; i++ {
					for filedb.findDBInfoNumber(fileIndex, curVersionNumber, targetNode) != -1 {
						targetNode = findNextHop(targetNode, membership)
					}
					log.Println("Copy from:", oneStoredNode, " to:", targetNode, " about:", file.sdfsName)
					portNumber := strings.Split(oneStoredNode, "|")
					filePath := "/tmp/sdfs/sdfs-" + portNumber[1] + "/" + version.storedName
					filedb.StoreSingleFile(masterID, filePath, file.sdfsName, version.storedName, oneStoredNode, targetNode)
				}

			}

		}
	}
}

func (filedb *FileDB) DBHandleOldFile(oldLocalStorageString string, storedID string) {
	oldLocalStorage := UnjsonLocalStorage([]byte(oldLocalStorageString))
	for _, oldLocalEntry := range oldLocalStorage {
		fmt.Println(oldLocalEntry.OriginName, ",storedName:", oldLocalEntry.StoredName)
		fileIdx := filedb.findDBFileByName(oldLocalEntry.OriginName)
		curVersions := filedb.findDBFileVersion(fileIdx, oldLocalEntry.StoredName)
		if filedb.findDBInfoNumber(fileIdx, curVersions, storedID) == -1 {
			filedb.addStoredNode(oldLocalEntry.OriginName, oldLocalEntry.StoredName, storedID)
		}
	}
}

//outstate 1 measn file exist, 0 means not exist
// func (filedb *FileDB) DBACKFile(sdfsName, storedName, sourceID, remoteState string) {
// 	fileIdx := filedb.findDBFileByName(sdfsName)
// 	storedVersionNumber := filedb.findDBFileVersion(fileIdx, storedName)
// 	InfoNumber := filedb.findDBInfoNumber(fileIdx, storedVersionNumber, sourceID)

// 	curState := (*filedb)[fileIdx].versions[storedVersionNumber].Info[InfoNumber].storedState
// 	if remoteState == "1" {
// 		(*filedb)[fileIdx].versions[storedVersionNumber].Info[InfoNumber].storedState = 1
// 	} else if curState == 0 && remoteState == "0" {
// 		(*filedb)[fileIdx].versions[storedVersionNumber].Info[InfoNumber].storedState = -1
// 	} else if curState == -1 && remoteState == "0" {
// 		filedb.deleteDBStoredNode(fileIdx, storedVersionNumber, InfoNumber)
// 	}
// }

/**
	deleted message handling
**/
func sendDeleteRequest(storedName, dstNode, masterID string) {
	dstMsg := com.Msg{com.DELETEREQUEST, masterID, storedName}
	com.SendMsg(dstMsg, dstNode, 0)
}
func sendFailedDeleteRequest(dstNode, masterID string) {
	dstMsg := com.Msg{com.DELETEREQUEST, masterID, "missing"}
	com.SendMsg(dstMsg, dstNode, 0)
}

func (filedb *FileDB) DBDeleteFile(source, masterID, sdfsName string) {
	fileIdx := filedb.findDBFileByName(sdfsName)
	if fileIdx == -1 {
		log.Println("file named:", sdfsName, " not exist in databse")
		sendFailedDeleteRequest(source, masterID)
		return
	} else {
		for _, version := range (*filedb)[fileIdx].versions {
			for _, infos := range version.Info {
				log.Println("send delete message about:", version.storedName)
				sendDeleteRequest(version.storedName, infos.storedNode, masterID)
			}
		}
		*filedb = append((*filedb)[:fileIdx], (*filedb)[fileIdx+1:]...)
	}

}
func (filedb *FileDB) DBDeleteFileByNode(deletedNode string) {

	db := *filedb
	for fileIdx, file := range db {
		for versionIDx, version := range file.versions {
			for storedIdx, entry := range version.Info {
				if entry.storedNode == deletedNode {
					filedb.deleteDBStoredNode(fileIdx, versionIDx, storedIdx)
					log.Println("delete file stored in node:", deletedNode, " the filename is:", file.sdfsName)
				}
			}
		}
	}

}

func findNextHop(nodeID string, membership []string) string {
	dst := membership[0]

	for i := range membership {
		if nodeID == membership[i] {
			index := (i + 1) % len(membership)
			dst = membership[index]
			break
		}
	}
	return dst
}
