package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"

	"./leader"

	"./config"

	"./com"
)

func setupNodeID(ip string, port string) {
	t := time.Now()
	timeStamp := t.Format("15:04:05.000")
	NodeID = ip + "|" + port + "|" + timeStamp
}

/**
	if an elemetn is in a slice string
**/
func contains(list []string, e string) int {
	if len(list) < 1 {
		return -1
	}
	for i, a := range list {
		if a == e {
			return i
		}
	}
	return -1
}

/**
	tools for introducer and receiver for the membership and membership list
**/
func jsonPing() string {
	jsonPing, err := json.Marshal(membership)
	if err != nil {
		log.Println(err, ", have problem json membership")
	}
	return string(jsonPing)
}
func unjsonPing(input []byte) {
	input = bytes.Trim(input, "\x00")
	membership = []string{}
	err := json.Unmarshal(input, &membership)

	if err != nil {
		log.Println(err, ", have problem unmarshal membership")
	}
}

/**
	tools for node and introducer maintain their pingNodes
**/
func updataMembershipList(memberID string, isAdd bool) (err error) {
	if memberID == NodeID {
		return nil
	}
	if isAdd {
		if len(pingNodes) < 5 {
			pingNodes[memberID] = time.Now().String() // add to membershipList
		}
	} else {
		if _, ok := pingNodes[memberID]; ok {
			delete(pingNodes, memberID)
			// ask for UPDATE from at most 5  members in the membership
			if len(pingNodes) < 1 {
				pingNodes = make(map[string]string)
				askpingNodes()
			}
		} else {
			return errors.New("err in delete " + memberID)
		}
	}
	return nil
}

func jsonpingNodes(partMember map[string]string) string {
	jsonList, err := json.Marshal(partMember)
	if err != nil {
		log.Println(err, ", have problem json membership")
	}
	return string(jsonList)
}
func unjsonpingNodes(input []byte) {
	input = bytes.Trim(input, "\x00")
	pingNodes = make(map[string]string)
	err := json.Unmarshal(input, &pingNodes)

	if err != nil {
		log.Println(err, ", have problem unmarshal membership")
	}
}
func distribuetmembership(config config.Config) {
	if config.IsIntro == false {
		membership = []string{}
	}

	for _, host := range membership {
		com.SendMsg(com.Msg{com.MEMBERSHIP, NodeID, jsonPing()}, host, falseRate)
	}
}
func distributepingNodes() {
	membershipLen := len(membership)

	// this host is destination
	for _, host := range membership {
		var tmppingNodes map[string]string
		tmppingNodes = make(map[string]string)
		var i int
		if membershipLen <= 5 {
			for _, loopHost := range membership {
				if loopHost != host { // destion membership will not cover itself
					tmppingNodes[loopHost] = time.Now().String()
				}
			}
		} else {
			counter := 0

			for i = distributePointer; ; i++ {
				tmpi := i
				if tmpi >= membershipLen {
					tmpi = tmpi % membershipLen
				}
				if membership[tmpi] == host {
					continue
				}

				tmppingNodes[membership[tmpi]] = pingNodes[membership[tmpi]]
				counter++
				if counter == 5 {
					break
				}
			}

		}
		com.SendMsg(com.Msg{com.PINGNODES, NodeID, jsonpingNodes(tmppingNodes)}, host, falseRate)
		distributePointer = i
		if distributePointer >= membershipLen {
			distributePointer = distributePointer % membershipLen
		}

	}

}

/**
	tools for normal host to maintain their pingNodes
**/

func askpingNodes() {
	log.Println(NodeID, " have no members now and is asking member list from its neighbors")
	counter := 5
	for _, host := range membership {
		if host == NodeID {
			continue
		}
		com.SendMsg(com.Msg{com.ASKLIST, NodeID, "ASKID"}, host, falseRate)
		counter -= 1
		if counter == 5 {
			break
		}
	}
}
func deleteNode(host string) (err error) {
	err = updataMembershipList(host, false)
	for i, e := range membership {
		if e == host {
			membership = append(membership[:i], membership[i+1:]...)
			break
		}
	}
	return nil
}
func sendDeadMsg(deadNodeID string) {

	deadMsg := com.Msg{com.DELNODE, NodeID, deadNodeID}
	delete(lastAck, deadNodeID) // no need to lock, since we are inside lock from above.
	deleteNode(deadNodeID)
	floodToMember(deadMsg)

	if deadNodeID == MasterID {
		log.Println("The leader is dead! I saw it with my own eyes!")
		leader.SendElectionMessage(NodeID, membership)
	}

}

//Sends Pings to the nodes its montering.
func startPingACK(config config.Config) {
	for {
		for host := range pingNodes {
			pingMsg := com.Msg{com.PING, NodeID, ""}
			com.SendMsg(pingMsg, host, falseRate)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func startJoin(config config.Config) {

	go func() {
		if !config.IsIntro {
			isJoined = false
			for isJoined == false {
				sendJoinMessage(config)
				time.Sleep(5 * time.Second)
			}
			//start sending pings
			go startPingACK(config)
			go checkLastAck()
		}
	}()

}

func sendJoinMessage(config config.Config) {
	conn, err := net.Dial("udp", config.Introducer)
	defer conn.Close()
	if err != nil {
		log.Fatal(err, " Could not connect to introducer, nothing we can do now.")
	}

	joinMessage := com.Msg{MsgType: com.JOIN, MsgSourceID: NodeID, MsgContent: "JOIN"}
	conn.Write(com.JsonToBytes(joinMessage))
}

func floodToMember(message com.Msg) (err error) {
	for member := range pingNodes {
		com.SendMsg(message, member, falseRate)
	}
	return nil
}

func floodToAll(message com.Msg) (err error) {
	for _, member := range membership {
		com.SendMsg(message, member, falseRate)
	}
	return nil
}

/**
	functions for file OP
**/
