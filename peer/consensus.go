package peer

import (
	"fmt"
	"log"

	"golang.org/x/sync/errgroup"
)

type WorkerState struct {
	Value int
}

func InitState(w *Worker) WorkerState {
	return WorkerState{0}
}

func (s *WorkerState) String() string {
	return fmt.Sprintf("Value: %d", s.Value)
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

type WorkerLog struct {
	Operation string
	Value     int
}

type RequestLogArgs struct{}

type RequestLogReply struct {
	Logs []*WorkerLog
}

func (w *Worker) RequestLog(args RequestLogArgs, reply *RequestLogReply) error {
	w.LockMutex()
	defer w.UnlockMutex()

	reply.Logs = w.Logs

	return nil
}

type UpdateStateArgs struct {
	Operation string
	Value     int
}

type UpdateStateReply struct {
	Before WorkerState
	After  WorkerState
}

func (w *Worker) UpdateState(args UpdateStateArgs, reply *UpdateStateReply) error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		return w.UpdateStateWithoutSync(args, reply)
	})

	for k := range w.ConnectedPeers() {
		k := k

		eg.Go(func() error {
			tmp := UpdateStateReply{}
			if err := w.RemoteCall(k, "Worker.UpdateStateWithoutSync", args, &tmp); err != nil {
				return err
			}

			return nil
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
