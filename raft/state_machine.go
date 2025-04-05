package raft

import (
	"log"
	"strings"
)

func (n *Node) ApplyToStateMachine(command string) {
	parts := strings.Fields(command)
	if len(parts) == 3 && parts[0] == "SET" {
		key := parts[1]
		value := parts[2]
		n.StateMachine[key] = value
		log.Printf("🧠 Applied to state machine: SET %s = %s", key, value)
	}
}
