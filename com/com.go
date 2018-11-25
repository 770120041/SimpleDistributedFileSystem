package com

import (
	"log"
	"net"
	"strings"
)

//Msg contains everything the protcol needs to know
type Msg struct {
	MsgType     int    `json:"msgType"`
	MsgSourceID string `json:"sourceID"`
	MsgContent  string `json:"content"`
}

// The types used in the Msg, part of the protocol specification
const (
	NONE = iota
	JOIN
	JOINACK
	UPDATE
	DELNODE
	PING
	ACK
	PINGNODES
	MEMBERSHIP
	ASKLIST

	//LS command
	LSREQUEST
	LSRESPONSE

	//file op Msg

	STOREFILE
	SENDREQUEST
	GETFILE
	GETRESPOND

	DELETEREQUEST
	DELETEFILE
	DELETEACK

	DBASKFILE
	DBASKRESPOND

	DBSYNC
	//election Msg
	ELECTION
	ELECTED
)

var sendPackages = 0

//SendMsg send message to a destination
func SendMsg(message Msg, dstID string, falseRate int) (err error) {
	dstInfo := strings.Split(dstID, "|")
	conn, err := net.Dial("udp", dstInfo[0]+":"+dstInfo[1])
	defer conn.Close()
	if err != nil {
		log.Println("Could not open udp socket")
		return err
	}

	// ranNumber := rand.Intn(99)
	// // discard the message due to false positive
	// if ranNumber+falseRate >= 100 {
	// 	log.Println("Dropped package. Total packages send", sendPackages)
	// 	return nil
	// }

	n, err := conn.Write(JsonToBytes(message))
	if n == 0 {
		log.Println("Did not write anything..")
	}
	if err != nil {
		log.Println(err)
		return err
	}

	sendPackages++
	return nil
}

//ParseMsg reads the incoming packet and turns it into a Msg
func ParseMsg(packet net.PacketConn) (message Msg) {
	buffer := make([]byte, 1024)
	n, _, err := packet.ReadFrom(buffer)
	if err != nil {
		log.Println(err)
		return Msg{NONE, "", ""} // err
	}
	if n == 0 {
		log.Println("No bytes received")
		return Msg{NONE, "", ""} // err
	}
	message = BytesToJson(buffer)
	return message
}
