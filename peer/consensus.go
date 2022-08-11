package peer

import (
	"log"
)

type HeartBeatArgs struct {
	Leader string
	Term   int

	Entry        WorkerLogs
	PrevLogIndex int
	LeaderCommit int
}

type HeartBeatReply struct {
	Updated bool
}

func (w *Worker) HeartBeat(args HeartBeatArgs, reply *HeartBeatReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	if args.Term < w.Term() {
		return ErrNotLeader
	}

	w.SetLeader(args.Leader)
	w.SetTerm(args.Term)
	w.ResetVoteTimer()

	if w.logs.find(args.PrevLogIndex) == nil {
		reply.Updated = false
		return nil
	}

	for _, v := range args.Entry {
		if w.logs.find(v.Index) != nil {
			continue
		}

		if err := w.AddLog(v); err != nil {
			return err
		}
	}

	w.logs.sort()

	if args.LeaderCommit > w.commitIndex {
		w.commitIndex = args.LeaderCommit

		if w.logs[len(w.logs)-1].Index < w.commitIndex {
			w.commitIndex = w.logs[len(w.logs)-1].Index
		}
	}

	for k := range w.nextIndices {
		w.nextIndices[k] = w.commitIndex
	}

	reply.Updated = true

	return nil
}

type VoteArgs struct {
	Leader string
	Term   int
}

type VoteReply struct {
	IfAccept bool
}

func (w *Worker) Vote(args VoteArgs, reply *VoteReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	if args.Term <= w.Term() {
		log.Printf("reject vote from %s: sent: %d, current: %d", args.Leader, args.Term, w.Term())

		reply.IfAccept = false
		return nil
	}

	log.Printf("accept vote from %s: sent: %d, current: %d", args.Leader, args.Term, w.Term())

	w.SetLeader(args.Leader)
	w.SetTerm(args.Term)
	w.ResetVoteTimer()

	reply.IfAccept = true
	return nil
}

type RequestStateArgs struct{}

type RequestStateReply struct {
	State WorkerState
}

func (w *Worker) RequestState(args RequestStateArgs, reply *RequestStateReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	reply.State = w.State()

	log.Println(reply.State.String())

	return nil
}

type RequestLogArgs struct{}

type RequestLogReply struct {
	Logs []*WorkerLog
}

func (w *Worker) RequestLog(args RequestLogArgs, reply *RequestLogReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	reply.Logs = w.Logs()

	return nil
}

type UpdateStateArgs struct {
	Operator string
	Operand  int
}

type UpdateStateReply struct {
	Before WorkerState
	After  WorkerState
}

func (w *Worker) UpdateState(args UpdateStateArgs, reply *UpdateStateReply) error {
	if w.Leader() != w.Name() {
		log.Printf("proxying RPC to leader(%s)", w.Leader())

		err := w.RemoteCall(w.Leader(), "Worker.UpdateState", args, &reply)
		if err != nil {
			return err
		}

		return nil
	}

	w.LockMutex()
	defer w.UnlockMutex()

	reply.Before = w.State()
	if err := w.AddLog(&WorkerLog{
		Index:    w.logs[len(w.logs)-1].Index + 1,
		Operator: args.Operator,
		Operand:  args.Operand,
	}); err != nil {
		return err
	}
	reply.After = w.State()

	log.Println(reply.Before.String(), "|", reply.After.String())

	return nil
}

type RequestLeaderArgs struct{}

type RequestLeaderReply struct {
	Leader string
}

func (w *Worker) RequestLeader(args RequestLeaderArgs, reply *RequestLeaderReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	reply.Leader = w.Leader()

	return nil
}
