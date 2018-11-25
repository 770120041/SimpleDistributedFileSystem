package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"./transfer"

	"./file"

	"./leader"

	"./config"

	"./com"
)

var pingNodes map[string]string //the members this node will ping
var membership []string         //all the nodes in the group

//Keep track on slow pings
var lastAck = make(map[string]time.Time)
var lastAckLock = &sync.Mutex{}

//Node ID of itself, contains IP and port
var NodeID string
var distributePointer int //for introducer
var isJoined bool
var isIntro bool
var inElection bool
var lastElectionInit string

var falseRate int = 0

var appConf config.Config

//DataStorage sections
var waitNewMaster bool
var filedb file.FileDB
var localStorage file.FileStorage

//Election Sections
var MasterID string = "No master assigned"

func main() {
	// set up memberlist and pinglist
	pingNodes = make(map[string]string)
	distributePointer = 0
	appConf = config.SetupFlags()

	setupNodeID(appConf.Ip, appConf.Port)
	membership = append(membership, NodeID)

	log.Println("Node starting at: " + NodeID)
	file.CreateSdfsFolder(appConf.Port)

	go transfer.StartTransferServer(appConf.Port, &localStorage)
	//start listening for incoming messages
	go startListening(appConf)
	//set up file storage

	if appConf.IsIntro {
		updataMembershipList(NodeID, true)
		isIntro = true
		//start sending pings
		go startPingACK(appConf)
		go checkLastAck()
	} else {
		startJoin(appConf)
	}

	// if isIntro {
	// 	MasterID = NodeID
	// }
	//small shell
	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			texts := strings.Split(scanner.Text(), " ")
			text := texts[0]
			if text == "leave" {
				floodToMember(com.Msg{com.DELNODE, NodeID, NodeID})
				pingNodes = make(map[string]string)
				membership = []string{}
				os.Exit(0)
			} else if text == "ping" {
				for member := range pingNodes {
					fmt.Println(member)
				}
			} else if text == "master" {
				fmt.Println(MasterID)
			} else if text == "id" {
				fmt.Println(NodeID)
			} else if text == "intro" {
				fmt.Println("The introducer is", appConf.Introducer)
			} else if text == "join" {
				//startJoin(appConf)
			} else if text == "sync" {
				filedb.DBStoreUpdate(MasterID, membership, "REGULAR")
			} else if text == "membership" {
				for _, ping := range membership {
					fmt.Println(ping)
				}
			} else if text == "put" {
				if len(texts) != 3 {
					fmt.Println("put filename sdfs ")
					continue
				}

				filePath := texts[1]
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					fmt.Println("file does not exist")
					continue
				}
				sdfsFileName := texts[2]
				com.SendMsg(com.Msg{com.STOREFILE, NodeID, filePath + ">>" + sdfsFileName}, MasterID, 0)

			} else if text == "store" {
				localStorage.ShowLocalFile()
			} else if text == "delete" {
				if len(texts) != 2 {
					fmt.Println("delete sdfsNames")
					continue
				}
				sdfsName := texts[1]
				com.SendMsg(com.Msg{com.DELETEFILE, NodeID, sdfsName}, MasterID, 0)
			} else if text == "ls" {
				if len(texts) != 2 {
					fmt.Println("ls sdfsNames")
					continue
				}
				sdfs := texts[1]
				com.SendMsg(com.Msg{com.LSREQUEST, NodeID, sdfs}, MasterID, 0)
			} else if text == "showdb" {
				if NodeID == MasterID {
					filedb.ShowDB()
				}
			} else if text == "get" {
				if len(texts) != 3 {
					fmt.Println("get sdfs localNmae")
					continue
				}
				sdfsName := texts[1]
				localName := texts[2]
				com.SendMsg(com.Msg{com.GETFILE, NodeID, sdfsName + ">>" + localName + ">>" + "-1"}, MasterID, 0)

			} else if text == "get-versions" {
				if len(texts) != 4 {
					fmt.Println("get-versions sdfs num-versions localName")
					continue
				}
				sdfsName := texts[1]
				numVersions := texts[2]
				localName := texts[3]

				com.SendMsg(com.Msg{com.GETFILE, NodeID, sdfsName + ">>" + localName + ">>" + numVersions}, MasterID, 0)
				//askMainFirst
				// transfer.DownloadData(localName, sdfsName, MasterID)

			}
		}
		if scanner.Err() != nil {
			log.Println(scanner.Err())
		}
	}()
	select {}

}

//checkLastAck will check the lastest ack from members, if to old it means a node is dead.
func checkLastAck() {
	for {
		lastAckLock.Lock()
		for id, lastAckTime := range lastAck {
			if time.Now().Sub(lastAckTime) > 1800*time.Millisecond {
				log.Println(id, "is not responding")
				sendDeadMsg(id)
			}
		}
		lastAckLock.Unlock()
		time.Sleep(1 * time.Second)
	}
}

func startListening(config config.Config) {
	packet, err := net.ListenPacket("udp", ":"+config.Port)
	defer packet.Close()
	if err != nil {
		log.Fatal(err)
	}

	for {
		message := com.ParseMsg(packet)
		switch message.MsgType {
		case com.NONE:
			log.Println("No message received, have err parsing or transmission")
		case com.JOIN:
			log.Println(message.MsgSourceID, "wants to JOIN")
			newMemberID := message.MsgSourceID

			if _, ok := pingNodes[newMemberID]; !ok {
				if len(membership) == 1 && MasterID == "No master assigned" {
					log.Println("The first to join will become Master!")
					MasterID = newMemberID

				}

				//update membership list and membership
				updataMembershipList(newMemberID, true)
				membership = append(membership, newMemberID)
				//sent out membership to new member
				joinACKMsg := com.Msg{com.JOINACK, NodeID, MasterID}
				err = com.SendMsg(joinACKMsg, newMemberID, falseRate)
				if err != nil {
					log.Fatal(err, ",send message back failed")
				}
				//pingNodes Message
				distribuetmembership(config)
				distributepingNodes()

			} else {
				log.Println("FATAL ERROR! ", NodeID, "  alread in membership list at:", pingNodes[newMemberID])
			}

			//update each time a new node

			com.SendMsg(com.Msg{com.DBSYNC, NodeID, "REGULAR"}, MasterID, 0)

		case com.JOINACK:
			log.Println("Joined group acknowledged by the introducer")
			log.Println("Master id is", message.MsgContent)
			MasterID = message.MsgContent
			isJoined = true
			//send by introducer
		case com.MEMBERSHIP:
			if NodeID != message.MsgSourceID {
				log.Println("Receive membership from introducer")
				lastAckLock.Lock()
				lastAck = make(map[string]time.Time)
				lastAckLock.Unlock()
				unjsonPing([]byte(message.MsgContent))
			}

		case com.PINGNODES:
			if NodeID != message.MsgSourceID {
				log.Println("Receive pingNodes from introducer")
				lastAckLock.Lock()
				lastAck = make(map[string]time.Time)
				lastAckLock.Unlock()
				unjsonpingNodes([]byte(message.MsgContent))
			}

		case com.UPDATE:
			if _, ok := pingNodes[message.MsgContent]; !ok {
				log.Println("Incoming UPDATE From:", message.MsgSourceID, ",about:", message.MsgContent) // if not known by itself, update and flood
				if index := contains(membership, message.MsgContent); index == -1 {
					membership = append(membership, message.MsgContent)
					//floodToMember(com.Msg{com.UPDATE, NodeID, message.MsgContent})
				}

				updataMembershipList(message.MsgContent, true)
			}

		case com.DELNODE:

			deleteNodeID := message.MsgContent

			if index := contains(membership, deleteNodeID); index != -1 {

				if deleteNodeID == MasterID {
					log.Println("The leader is dead! I heard about it!")
					leader.SendElectionMessage(NodeID, membership)
				}

				log.Println(message.MsgSourceID, "SAID DELETE", deleteNodeID) //if not known by itself, update and flood
				membership = append(membership[:index], membership[index+1:]...)
				updataMembershipList(deleteNodeID, false)
				floodToMember(com.Msg{com.DELNODE, NodeID, deleteNodeID})
			}
			if deleteNodeID == MasterID {
				log.Println("Master dies! we need a new master to gather info!")
				waitNewMaster = true

			} else if NodeID == MasterID {
				log.Println("DB need to update because:", deleteNodeID, " died")
				filedb.DBStoreUpdate(MasterID, membership, deleteNodeID)
			}
			lastAckLock.Lock()
			delete(lastAck, message.MsgContent)
			lastAckLock.Unlock()

		case com.PING:
			ackMsg := com.Msg{com.ACK, NodeID, "ACK"}
			com.SendMsg(ackMsg, message.MsgSourceID, falseRate)

		case com.ACK:
			lastAckLock.Lock()
			lastAck[message.MsgSourceID] = time.Now()
			lastAckLock.Unlock()
		case com.ASKLIST: // send out one member update the asker
			if len(pingNodes) <= 1 {
				com.SendMsg(com.Msg{com.UPDATE, NodeID, NodeID}, message.MsgSourceID, falseRate)
			}
			for key, _ := range pingNodes {
				if key != message.MsgSourceID {
					com.SendMsg(com.Msg{com.UPDATE, NodeID, key}, message.MsgSourceID, falseRate)
					break
				}
			}
		case com.ELECTION:
			if inElection {
				if !leader.ValidateInitiator(message.MsgContent, lastElectionInit) {
					log.Println("Ignoring incoming election")
					break
				}
			}
			log.Println("Incoming election message from", message.MsgSourceID, message.MsgContent)
			isValidLeader, isLeader, initiator := leader.ValidateElectionMessage(message.MsgContent, NodeID)
			log.Println("is this a valid leader", isValidLeader, "is this the leader", isLeader, message.MsgContent)
			if isLeader {
				leader.SendElectedMessage(NodeID, NodeID, membership)
				break
			}

			if isValidLeader {
				leader.SendElectionMessage(NodeID, membership)
			} else {
				leader.ForwardElectionMessage(message.MsgContent, NodeID, membership)
			}
			inElection = true
			lastElectionInit = initiator

		//after being elected:send out DBASKFILE by master
		case com.DBASKFILE:
			if len(localStorage) > 0 {
				dstMsg := com.Msg{com.DBASKRESPOND, NodeID, file.JsonLocalStorage(localStorage)}
				fmt.Println("node send out:", file.JsonLocalStorage(localStorage))
				com.SendMsg(dstMsg, message.MsgSourceID, 0)
			}
		//other node return to dbASKRESPOND
		case com.DBASKRESPOND:
			filedb.DBHandleOldFile(message.MsgContent, message.MsgSourceID)
		case com.DBSYNC:
			if MasterID == NodeID {
				filedb.DBStoreUpdate(MasterID, membership, message.MsgContent)
			} else {
				log.Println("ask sync to a none master node! from", message.MsgSourceID)
			}
		case com.ELECTED:
			log.Println("Hurray! we have a new leader!", message.MsgContent)
			inElection = false
			MasterID = message.MsgContent
			if NodeID == message.MsgContent {
				//After being elected, send DBASKFILE
				dstMsg := com.Msg{com.DBASKFILE, NodeID, "ASKFILE"}
				floodToAll(dstMsg)
				break
			} else {
				leader.SendElectedMessage(message.MsgContent, NodeID, membership)
			}

		//used for master
		case com.STOREFILE:
			if MasterID == NodeID {
				filenames := strings.Split(message.MsgContent, ">>")
				filedb.StoreNewFile(MasterID, filenames[0], filenames[1], message.MsgSourceID, membership)
			} else {
				log.Println("Sending store msg to a none master node! from", message.MsgSourceID)
			}

		case com.SENDREQUEST:
			filePath, sdfsFilename, dstFilename, _, dstID := file.ParseStoreMsg(message.MsgContent)
			go transfer.SendData(filePath, sdfsFilename, dstFilename, dstID)

		case com.GETFILE:
			contents := strings.Split(message.MsgContent, ">>")
			sdfsName, localName, dstVersion, sourceNode := contents[0], contents[1], contents[2], message.MsgSourceID
			log.Println("receive GETFILE for file", sdfsName, " with new ID", sourceNode)
			filedb.GetDBFileRequest(MasterID, sdfsName, localName, dstVersion, sourceNode)
		case com.GETRESPOND:
			log.Println("respond from getfile:")
			if message.MsgContent == "missing" {
				log.Println("the file is not available")
				continue
			}

			contents := strings.Split(message.MsgContent, ">>")
			storedNode, storedName, localName := contents[0], contents[1], contents[2]
			go transfer.DownloadData(localName, storedName, storedNode)

		case com.DELETEREQUEST:
			localName := message.MsgContent
			if localName == "missing" {
				log.Println("the file is not available")
				continue
			}

			log.Println("receive delete request about storedName:", localName, " in node:", NodeID)
			localStorage.RemoveFileStorage(localName)
			go file.DeleteFile(localName)
		case com.DELETEFILE:
			if NodeID == MasterID {
				sdfsName := message.MsgContent
				log.Println("Deletefile name:", sdfsName, " from:", message.MsgSourceID)
				filedb.DBDeleteFile(message.MsgSourceID, MasterID, sdfsName)
			} else {
				log.Println("Sending delete file  a none master node! from", message.MsgSourceID)
			}

		case com.LSREQUEST:
			log.Println(message.MsgSourceID, "wants to know who stores", message.MsgContent)
			requester := message.MsgSourceID
			sdfs := message.MsgContent
			allNodes := filedb.GetNodesThatStore(sdfs)
			nodes := strings.Join(allNodes, ">>")
			com.SendMsg(com.Msg{com.LSRESPONSE, NodeID, sdfs + "<<" + nodes}, requester, 0)

		case com.LSRESPONSE:
			message := strings.Split(message.MsgContent, "<<")
			sdfs, nodes := message[0], message[1]
			allNodes := strings.Split(nodes, ">>")
			fmt.Println("Nodes that stores sdfs file", sdfs)
			for _, v := range allNodes {
				fmt.Println(v)
			}

		default:
			log.Println("This is an unknown message")
		}

	}

}
