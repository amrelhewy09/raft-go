package services

import (
	"github.com/amrelhewy/raft-go/raft"
)

type RaftService struct {
	*raft.Node
}

func (s *RaftService) RequestVote(args *raft.RequestVoteArgs, reply *raft.RequestVoteReply) error {
	return s.Node.RequestVote(args, reply)
}

func (s *RaftService) AppendEntries(args *raft.AppendEntriesArgs, reply *raft.AppendEntriesReply) error {
	return s.Node.AppendEntries(args, reply)
}

func (s *RaftService) HandleClientCommand(args *raft.ClientCommandArgs, reply *raft.ClientCommandReply) error {
	return s.Node.HandleClientCommand(args, reply)
}

func (s *RaftService) GetValue(args *raft.GetValueArgs, reply *raft.GetValueReply) error {
	return s.Node.GetValue(args, reply)
}
