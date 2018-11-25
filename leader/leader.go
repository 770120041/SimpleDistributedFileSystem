package leader

import (
	"log"
	"strconv"
	"strings"

	"../com"
)

func ValidateInitiator(message, currentInitiator string) bool {
	electionData := strings.Split(message, "->")
	initiator := electionData[0]
	pruposedInitiator, err := strconv.Atoi(initiator)
	if err != nil {
		log.Fatal("proposed initiator value is not an integer...")
	}

	currInit, err := strconv.Atoi(currentInitiator)
	if err != nil {
		log.Fatal("current initiator value is not integer")
	}

	if currInit > pruposedInitiator {
		return false
	}
	return true
}

func ValidateElectionMessage(message, node string) (bool, bool, string) {
	electionData := strings.Split(message, "->")
	initiator := electionData[0]
	proposedLeader, err := strconv.Atoi(electionData[1])
	if err != nil {
		log.Fatal("proposed leader value is not an integer...")
	}

	currentValue := calculateLeaderValue(node)
	log.Println("Current value", currentValue, "proposed value", proposedLeader)
	if currentValue > proposedLeader {
		return true, false, initiator
	} else {
		if currentValue == proposedLeader {
			return false, true, initiator
		}
		return false, false, initiator
	}
}

func findNextHop(nodeID string, membership []string) string {
	dst := ""

	for i := range membership {
		if nodeID == membership[i] {
			index := (i + 1) % len(membership)
			dst = membership[index]
			break
		}
	}
	return dst
}

func ForwardElectionMessage(message, nodeID string, membership []string) {
	dst := findNextHop(nodeID, membership)
	electionMessage := com.Msg{com.ELECTION, nodeID, message}
	com.SendMsg(electionMessage, dst, 0)

}

func SendElectionMessage(nodeID string, membership []string) {
	ForwardElectionMessage(createElectionMessage(nodeID), nodeID, membership)
}

func SendElectedMessage(message, nodeID string, membership []string) {
	dst := findNextHop(nodeID, membership)
	electionMessage := com.Msg{com.ELECTED, nodeID, message}
	com.SendMsg(electionMessage, dst, 0)
}

func createElectionMessage(nodeId string) string {
	value := strconv.Itoa(calculateLeaderValue(nodeId))
	el := value + "->" + value
	return el
}

func calculateLeaderValue(nodeId string) int {
	ipAndPort := strings.Split(nodeId, "|")

	ip := ipAndPort[0]
	port, err := strconv.Atoi(ipAndPort[1])

	if err != nil {
		log.Fatal("Node has inccorect id")
	}

	leaderValue := port

	ipParts := strings.Split(ip, ".")

	for _, v := range ipParts {
		num, _ := strconv.Atoi(v)
		leaderValue += num

	}

	return leaderValue
}

func IDtoIP(nodeId string) string {
	ipAndPort := strings.Split(nodeId, "|")
	if len(ipAndPort) < 2 {
		log.Fatal("Just error")
	}
	ip := ipAndPort[0]
	port := ipAndPort[1]
	return ip + ":" + port
}
