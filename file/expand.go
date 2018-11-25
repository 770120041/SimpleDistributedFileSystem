package file

import "fmt"

func (filedb *FileDB) expandStorage(fileIndex int, curVersions int, targetNode, masterID string, membership []string, filePath, sdfsFileNamestring, dstFileName, srcID string) {
	expandReplicaNumber := 1

	versionNumber := filedb.findDBFileVersion(fileIndex, sdfsFileNamestring)
	if versionNumber == -1 {
		if len(membership) < 4 {
			expandReplicaNumber = len(membership)
		} else {
			expandReplicaNumber = 4
		}
	} else {
		existReplicaNumber := len((*filedb)[fileIndex].versions[versionNumber].Info)

		expandReplicaNumber = 4 - existReplicaNumber
		if len(membership)-existReplicaNumber < expandReplicaNumber {
			expandReplicaNumber = len(membership) - existReplicaNumber
		}
		fmt.Println("exist stored:", existReplicaNumber, " len membership:", len(membership))
	}
	fmt.Println("final decied to store :", expandReplicaNumber, " of replicas")
	//send storage request to one random node and 3 subsequent nodes

	for i := 0; i < expandReplicaNumber; i++ {
		// not stored replica files
		for filedb.findDBInfoNumber(fileIndex, curVersions, targetNode) != -1 {
			targetNode = findNextHop(targetNode, membership)
		}
		filedb.StoreSingleFile(masterID, filePath, sdfsFileNamestring, dstFileName, srcID, targetNode)
		targetNode = findNextHop(targetNode, membership)

	}
}
