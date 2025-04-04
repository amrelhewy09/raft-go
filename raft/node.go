package raft

import (
	"fmt"
	"log"
	"net/rpc"
	"sync"
	"time"
)

type State string

const (
	Follower  State = "follower"
	Candidate State = "candidate"
	Leader    State = "leader"
)

type LogEntry struct {
	Term    int
	Command interface{}
}

type Node struct {
	ID            string
	Addr          string
	mu            sync.Mutex
	CurrentTerm   int
	VotedFor      string
	State         State
	Peers         []string
	Log           []LogEntry
	electionTimer *time.Timer
	LeaderID      string
	LeaderAddr    string
	CommitIndex   int
}

func (n *Node) Ping(msg string, reply *string) error {
	fmt.Printf("[%s] got ping: %s\n", n.ID, msg)
	*reply = fmt.Sprintf("pong from %s", n.ID)
	return nil
}

func (n *Node) SendPing(peer string) {
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Printf("[%s] error connecting to %s: %v", n.ID, peer, err)
		return
	}
	defer client.Close()

	var reply string
	err = client.Call("Node.Ping", "ping from "+n.ID, &reply)
	if err != nil {
		log.Printf("[%s] error calling Ping on %s: %v", n.ID, peer, err)
		return
	}
	log.Printf("[%s] got reply from %s: %s", n.ID, peer, reply)
}
