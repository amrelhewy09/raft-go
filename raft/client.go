package raft

import (
	"log"
	"net/rpc"
	"sync"
)

type ClientCommandArgs struct {
	Command string
}

type ClientCommandReply struct {
	Result         string
	RedirectLeader string
	Err            string
}

type GetValueArgs struct {
	Key string
}

type GetValueReply struct {
	Value string
	Found bool
}

func (n *Node) GetValue(args *GetValueArgs, reply *GetValueReply) error {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	val, ok := n.StateMachine[args.Key]
	if ok {
		reply.Value = val
		reply.Found = true
	} else {
		reply.Found = false
	}
	return nil
}

func (n *Node) HandleClientCommand(args *ClientCommandArgs, reply *ClientCommandReply) error {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	if n.State != Leader {
		reply.Err = "not leader"
		reply.RedirectLeader = n.LeaderAddr
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
	go replicate(n, entry, prevIndex, prevTerm)

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
	var wg sync.WaitGroup
	for _, peer := range n.Peers {
		wg.Add(1)
		go func(peerAddr string) {
			defer wg.Done()
			client, err := rpc.DialHTTP("tcp", peerAddr)
			if err != nil {
				log.Printf("❌ Could not reach %s: %v", peerAddr, err)
				return
			}
			defer client.Close()

			var reply AppendEntriesReply
			err = client.Call("Node.AppendEntries", &replicateArgs, &reply)
			if err != nil {
				log.Printf("❌ AppendEntries RPC failed to %s: %v", peerAddr, err)
				return
			}

			if reply.Success {
				n.Mu.Lock()
				n.MatchIndex[peerAddr] = replicateArgs.PrevLogIndex + 1

				n.Mu.Unlock()
				log.Printf("✅ Log entry replicated to %s", peerAddr)

			} else {
				log.Printf("❌ Log rejected by %s — maybe inconsistency or stale term", peerAddr)
			}
		}(peer)
	}
	wg.Wait()
	go n.tryCommit(replicateArgs.PrevLogIndex + 1)
}
func (n *Node) tryCommit(entryIndex int) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	if n.Log[entryIndex].Term != n.CurrentTerm {
		return
	}

	count := 1 // include leader
	for _, idx := range n.MatchIndex {
		if idx >= entryIndex {
			count++
		}
	}

	total := len(n.Peers) + 1
	majority := total/2 + 1
	if count >= majority && n.CommitIndex < entryIndex {
		n.CommitIndex = entryIndex
		log.Printf("[%s] ✅ committed index %d!", n.ID, entryIndex)
	}
}
