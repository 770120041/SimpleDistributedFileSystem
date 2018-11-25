package com

import (
	"bytes"
	"encoding/json"
	"log"
)

//JsonToBytes marshal a message struct to an array of bytes.
func JsonToBytes(message Msg) []byte {
	messageMar, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err, "Malformed message")
	}
	return messageMar
}

func BytesToJson(input []byte) (message Msg) {
	input = bytes.Trim(input, "\x00")
	err := json.Unmarshal(input, &message)

	if err != nil {
		log.Fatal(err, " Have problem Unmarshall the message")
		return Msg{NONE, "", ""} // err
	}
	return message
}
