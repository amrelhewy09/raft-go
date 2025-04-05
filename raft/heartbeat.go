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
	n.Mu.Lock()
	defer n.Mu.Unlock()

	// log.Printf("[%s] 📬 received AppendEntries from leader %s (term %d) with %d entries",
	// n.ID, args.LeaderID, args.Term, len(args.Entries))

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

	n.LeaderID = args.LeaderID
	n.LeaderAddr = args.Addr
	n.ResetElectionTimer()

	if args.LeaderCommit > n.CommitIndex {
		n.CommitIndex = min(args.LeaderCommit, len(n.Log)-1)
		log.Printf("[%s] 📌 updated commit index to %d", n.ID, n.CommitIndex)
	}

	if args.PrevLogIndex >= 0 {
		if args.PrevLogIndex >= len(n.Log) {
			reply.Success = false
			reply.Term = n.CurrentTerm
			return nil
		}

		if n.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
			reply.Success = false
			reply.Term = n.CurrentTerm
			return nil
		}
	}

	for i, entry := range args.Entries {
		index := args.PrevLogIndex + 1 + i
		if index < len(n.Log) {
			continue
		}
		n.Log = append(n.Log, entry)
		log.Printf("[%s] 📥 appended log entry at index %d: %v", n.ID, index, entry.Command)
	}

	reply.Success = true
	reply.Term = n.CurrentTerm
	return nil
}

func (n *Node) StartHeartbeatTicker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		n.Mu.Lock()
		if n.State != Leader {
			n.Mu.Unlock()
			return
		}
		term := n.CurrentTerm
		n.Mu.Unlock()

		for _, peer := range n.Peers {
			go func(peer string) {
				client, err := rpc.DialHTTP("tcp", peer)
				if err != nil {
					return
				}
				defer client.Close()

				args := AppendEntriesArgs{
					Term:         term,
					LeaderID:     n.ID,
					Addr:         n.Addr,
					Entries:      []LogEntry{},
					LeaderCommit: n.CommitIndex,
				}
				var reply AppendEntriesReply
				client.Call("Node.AppendEntries", &args, &reply)
			}(peer)
		}

		<-ticker.C
	}
}
