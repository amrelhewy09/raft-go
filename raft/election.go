package raft

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

type RequestVoteArgs struct {
	Term        int
	CandidateID string
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

const (
	ElectionTimeoutMin = 150 * time.Millisecond
	ElectionTimeoutMax = 600 * time.Millisecond
)

func (n *Node) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if args.Term < n.CurrentTerm {
		reply.Term = n.CurrentTerm
		reply.VoteGranted = false
		return nil
	}

	if args.Term > n.CurrentTerm {
		n.CurrentTerm = args.Term
		n.VotedFor = ""
		n.State = Follower
	}

	if n.VotedFor == "" || n.VotedFor == args.CandidateID {
		fmt.Printf("[%s] 🔄 🔄 🔄 🔄 granting vote to %s in term %d\n", n.ID, args.CandidateID, args.Term)
		n.VotedFor = args.CandidateID
		n.CurrentTerm = args.Term
		reply.VoteGranted = true
	} else {
		fmt.Printf("[%s] 🔄 🔄 🔄 🔄 denying vote to %s in term %d\n", n.ID, args.CandidateID, args.Term)
		reply.VoteGranted = false
	}

	reply.Term = n.CurrentTerm
	return nil
}

func (n *Node) ResetElectionTimer() {
	if n.electionTimer != nil {
		n.electionTimer.Stop()
	}
	duration := time.Duration(rand.Intn(int(ElectionTimeoutMax-ElectionTimeoutMin))) + ElectionTimeoutMin

	n.electionTimer = time.AfterFunc(duration, func() {
		n.startElection()
	})
}

func (n *Node) startElection() {
	n.mu.Lock()
	n.State = Candidate
	n.CurrentTerm++
	n.VotedFor = n.ID
	term := n.CurrentTerm
	n.mu.Unlock()
	fmt.Printf("[%s] 💣 💣 💣 💣 starting election for term %d\n", n.ID, term)
	var votes int32 = 1
	var wg sync.WaitGroup
	for _, peer := range n.Peers {
		wg.Add(1)
		go func(peerAddr string) {
			defer wg.Done()
			args := RequestVoteArgs{
				Term:        term,
				CandidateID: n.ID,
			}
			var reply RequestVoteReply

			client, err := rpc.DialHTTP("tcp", peerAddr)
			if err != nil {
				return
			}
			defer client.Close()

			err = client.Call("Node.RequestVote", &args, &reply)
			if err != nil {
				fmt.Printf("[%s] 🔄 🔄 🔄 🔄 error sending request vote to %s in term %d: %v\n", n.ID, peerAddr, term, err)
			}

			if reply.VoteGranted {
				atomic.AddInt32(&votes, 1)
			}
		}(peer)
	}
	wg.Wait()
	log.Printf("[%s] 🎉 🎉 🎉 🎉 received %d votes for term %d", n.ID, votes, term)
	majority := (len(n.Peers)+1)/2 + 1

	if int(votes) >= majority {
		log.Printf("[%s] 🎉 became leader for term %d with %d votes", n.ID, term, votes)
		n.becomeLeader()
	} else {
		log.Printf("[%s] failed to become leader", n.ID)
		n.ResetElectionTimer()
	}
}

func (n *Node) becomeLeader() {

	n.mu.Lock()
	n.State = Leader
	n.mu.Unlock()

	go n.StartHeartbeatTicker()
}
