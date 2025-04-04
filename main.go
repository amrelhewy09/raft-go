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
		ID:          os.Getenv("ID"),
		Addr:        os.Getenv("ADDR"),
		CurrentTerm: 0,
		State:       raft.Follower,
		Peers:       getPeersFromEnv(),
		Log:         []raft.LogEntry{},
	}

	rpc.RegisterName("Node", &services.RaftService{Node: node})
	rpc.HandleHTTP()
	go func() {
		log.Printf("[%s] listening on %s\n", node.ID, node.Addr)
		log.Fatal(http.ListenAndServe(node.Addr, nil))
	}()

	time.Sleep(5 * time.Second)

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
