package raft

import (
	"log"
	"net/rpc"
)

type ClientCommandArgs struct {
	Command string
}

type ClientCommandReply struct {
	Result         string
	RedirectLeader string
	Err            string
}

func (n *Node) HandleClientCommand(args *ClientCommandArgs, reply *ClientCommandReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.State != Leader {
		reply.Err = "not leader"
		reply.RedirectLeader = n.LeaderID
		return nil
	}

	log.Printf("[%s] 💡 received command: %s", n.ID, args.Command)
	entry := LogEntry{Term: n.CurrentTerm, Command: args.Command}
	n.Log = append(n.Log, entry)
	lastIndex := len(n.Log) - 1
	prevIndex := lastIndex - 1
	prevTerm := 0
	if prevIndex >= 0 {
		prevTerm = n.Log[prevIndex].Term
	}
	replicate(n, entry, prevIndex, prevTerm)

	reply.Result = "command accepted"
	return nil
}

func replicate(n *Node, entry LogEntry, prevIndex int, prevTerm int) {
	replicateArgs := AppendEntriesArgs{
		Term:         n.CurrentTerm,
		LeaderID:     n.ID,
		Addr:         n.Addr,
		PrevLogIndex: prevIndex,
		PrevLogTerm:  prevTerm,
		Entries:      []LogEntry{entry},
		LeaderCommit: n.CommitIndex,
	}
	for _, peer := range n.Peers {
		go func(peerAddr string) {
			client, err := rpc.DialHTTP("tcp", peerAddr)
			if err != nil {
				log.Printf("❌ Could not reach %s: %v", peerAddr, err)
				return
			}
			defer client.Close()

			var reply AppendEntriesReply
			err = client.Call("RaftService.AppendEntries", &replicateArgs, &reply)
			if err != nil {
				log.Printf("❌ AppendEntries RPC failed to %s: %v", peerAddr, err)
				return
			}

			if reply.Success {
				log.Printf("✅ Log entry replicated to %s", peerAddr)
				// TODO: track acks and maybe commit if majority
			} else {
				log.Printf("❌ Log rejected by %s — maybe inconsistency or stale term", peerAddr)
			}
		}(peer)
	}

}
