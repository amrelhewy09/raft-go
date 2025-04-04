package raft

import (
	"log"
	"net/rpc"
	"time"
)

type AppendEntriesArgs struct {
	Term         int
	LeaderID     string
	Addr         string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (n *Node) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.Printf("[%s] 📬 received heartbeat from leader %s (term %d)", n.ID, args.LeaderID, args.Term)

	n.mu.Lock()
	defer n.mu.Unlock()

	if args.Term < n.CurrentTerm {
		reply.Term = n.CurrentTerm
		reply.Success = false
		return nil
	}

	if args.Term > n.CurrentTerm {
		n.CurrentTerm = args.Term
		n.State = Follower
		n.VotedFor = ""
	}

	n.LeaderID = args.Addr
	n.ResetElectionTimer()
	reply.Term = n.CurrentTerm
	reply.Success = true
	return nil
}

func (n *Node) StartHeartbeatTicker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		n.mu.Lock()
		if n.State != Leader {
			n.mu.Unlock()
			return
		}
		term := n.CurrentTerm
		n.mu.Unlock()

		for _, peer := range n.Peers {
			go func(peer string) {
				client, err := rpc.DialHTTP("tcp", peer)
				if err != nil {
					return
				}
				defer client.Close()

				args := AppendEntriesArgs{
					Term:     term,
					LeaderID: n.ID,
					Addr:     n.Addr,
				}
				var reply AppendEntriesReply
				client.Call("Node.AppendEntries", &args, &reply)
			}(peer)
		}

		<-ticker.C
	}
}
