package peer

import (
	"log"

	"golang.org/x/sync/errgroup"
)

type PingArgs struct {
	Leader string
	Term   int
}

type PingReply struct{}

func (w *Worker) Ping(args PingArgs, reply *PingReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	if args.Term < w.Term() {
		return ErrNotLeader
	}

	w.SetLeader(args.Leader)
	w.SetTerm(args.Term)
	w.ResetVoteTimer()

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

	eg := errgroup.Group{}

	eg.Go(func() error {
		return w.UpdateStateWithoutSync(args, reply)
	})

	for k := range w.ConnectedPeers() {
		k := k

		eg.Go(func() error {
			return w.RemoteCall(k, "Worker.UpdateStateWithoutSync", args, &UpdateStateReply{})
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) UpdateStateWithoutSync(args UpdateStateArgs, reply *UpdateStateReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	reply.Before = w.State()
	if err := w.AddLog(WorkerLog(args)); err != nil {
		return err
	}
	reply.After = w.State()

	log.Println(reply.Before.String(), reply.After.String())

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
