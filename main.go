package main

import (
	"log"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"time"

	"github.com/amrelhewy/raft-go/raft"
	"github.com/amrelhewy/raft-go/raft/services"
)

func main() {

	node := &raft.Node{
		ID:           os.Getenv("ID"),
		Addr:         os.Getenv("ADDR"),
		CurrentTerm:  0,
		State:        raft.Follower,
		Peers:        getPeersFromEnv(),
		Log:          []raft.LogEntry{},
		StateMachine: make(map[string]string),
		MatchIndex:   make(map[string]int),
		CommitIndex:  -1,
		LastApplied:  -1,
	}

	rpc.RegisterName("Node", &services.RaftService{Node: node})
	rpc.HandleHTTP()
	go func() {
		log.Printf("[%s] listening on %s\n", node.ID, node.Addr)
		log.Fatal(http.ListenAndServe(node.Addr, nil))
	}()

	time.Sleep(5 * time.Second)

	go func() {
		for {
			node.Mu.Lock()
			for node.LastApplied < node.CommitIndex {
				node.LastApplied++
				entry := node.Log[node.LastApplied]

				node.ApplyToStateMachine(entry.Command.(string))

				log.Printf("[%s] 🧠 applied index %d to state machine: %s", node.ID, node.LastApplied, entry.Command)
			}
			node.Mu.Unlock()

			time.Sleep(10 * time.Millisecond)
		}
	}()

	node.ResetElectionTimer()

	select {}
}

func getPeersFromEnv() []string {
	peersStr := os.Getenv("PEERS")
	if peersStr == "" {
		return []string{}
	}
	return strings.Split(peersStr, ",")
}
