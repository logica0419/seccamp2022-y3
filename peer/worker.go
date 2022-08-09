package peer

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type WorkerState struct {
	Term  int
	Value int
}

func (s *WorkerState) String() string {
	return fmt.Sprintf("Value: %d, Term: %d", s.Value, s.Term)
}

type WorkerLog struct {
	Operator string
	Operand  int
}

type Worker struct {
	name string
	node *Node

	mu sync.Mutex

	logs []*WorkerLog

	term   int
	leader string

	voteDuration time.Duration
	voteTimer    *time.Timer
}

type WorkerOption func(*Worker)

func NewWorker(name string) *Worker {
	w := new(Worker)
	w.name = name
	w.logs = []*WorkerLog{}
	return w
}

func (w *Worker) Name() string {
	return w.name
}

func (w *Worker) Addr() string {
	return w.node.Addr()
}

func (w *Worker) LockMutex() {
	w.mu.Lock()
}

func (w *Worker) UnlockMutex() {
	w.mu.Unlock()
}

func (w *Worker) Rand() *rand.Rand {
	return w.node.Rand()
}

func (w *Worker) LinkNode(n *Node) {
	w.node = n
}

func (w *Worker) Term() int {
	return w.term
}

func (w *Worker) Leader() string {
	return w.leader
}

func (w *Worker) SetLeader(leader string) {
	w.leader = leader
}

func (w *Worker) Logs() []*WorkerLog {
	return w.logs
}

func (w *Worker) AddLog(l WorkerLog) error {
	if l.Operator != "+" && l.Operator != "-" && l.Operator != "*" && l.Operator != "/" {
		return fmt.Errorf("invalid operator: %s", l.Operator)
	}

	w.logs = append(w.logs, &l)
	return nil
}

func (w *Worker) DeleteLastLog() error {
	if len(w.logs) == 0 {
		return fmt.Errorf("no logs to delete")
	}

	w.logs = w.logs[:len(w.logs)-1]
	return nil
}

func (w *Worker) State() WorkerState {
	temp := 0

	for _, v := range w.logs {
		switch v.Operator {
		case "+":
			temp += v.Operand
		case "-":
			temp -= v.Operand
		case "*":
			temp *= v.Operand
		case "/":
			temp /= v.Operand
		}
	}

	return WorkerState{temp, w.term}
}

func (w *Worker) PingTicker() {
	t := time.NewTicker(10 * time.Millisecond)

	for {
		<-t.C
		if w.Leader() != w.Name() {
			continue
		}

		eg := errgroup.Group{}
		for k := range w.ConnectedPeers() {
			k := k

			eg.Go(func() error {
				reply := PingReply{}

				err := w.RemoteCall(k, "Worker.Ping", PingArgs{w.Name(), w.Term()}, &reply)
				if err != nil {
					return err
				}
				if !reply.OK {
					return fmt.Errorf("already not leader")
				}

				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			w.LockMutex()
			w.SetLeader("")
			w.ResetVoteTimer()
			w.UnlockMutex()
		}
	}
}

func (w *Worker) VoteTimer() {
	w.voteDuration = (time.Duration(w.Rand().ExpFloat64()/50) + 10) * time.Millisecond
	w.voteTimer = time.NewTimer(w.voteDuration)

	for {
		<-w.voteTimer.C

		// TODO: Voteを実装
	}
}

func (w *Worker) ResetVoteTimer() {
	w.voteTimer.Reset(w.voteDuration)
}

func (w *Worker) Connect(name, addr string) (err error) {
	w.LockMutex()
	defer w.UnlockMutex()
	err = w.node.Connect(name, addr)
	if err != nil {
		return err
	}
	var reply RequestConnectReply
	err = w.RemoteCall(name, "Worker.RequestConnect", RequestConnectArgs{w.name, w.node.Addr()}, &reply)
	if err != nil {
		return err
	} else if !reply.OK {
		return fmt.Errorf("Connection request denied: [%s] %s", name, addr)
	}
	// for n, a := range reply.Peers {
	// 	if !w.node.IsConnectedTo(n) {
	// 		err = w.node.Connect(n, a)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }
	return nil
}

func (w *Worker) Stop() {
	w.node.Shutdown()
	w.node = nil
}

func (w *Worker) RemoteCall(name, method string, args any, reply any) error {
	return w.node.call(name, method, args, reply)
}

func (w *Worker) ConnectedPeers() map[string]string {
	return w.node.ConnectedNodes()
}

type RequestConnectArgs struct {
	Name string
	Addr string
}

type RequestConnectReply struct {
	OK    bool
	Peers map[string]string
}

func (w *Worker) RequestConnect(args RequestConnectArgs, reply *RequestConnectReply) error {
	reply.OK = false
	reply.Peers = make(map[string]string)
	w.LockMutex()
	defer w.UnlockMutex()
	err := w.node.Connect(args.Name, args.Addr)
	if err != nil {
		return err
	}
	reply.OK = true
	for name, addr := range w.node.ConnectedNodes() {
		reply.Peers[name] = addr
	}
	return nil
}

type RequestConnectedPeersArgs struct{}

type RequestConnectedPeersReply struct {
	Peers map[string]string
}

func (w *Worker) RequestConnectedPeers(args RequestConnectedPeersArgs, reply *RequestConnectedPeersReply) error {
	w.LockMutex()
	defer w.UnlockMutex()
	reply.Peers = make(map[string]string)
	for k, v := range w.ConnectedPeers() {
		reply.Peers[k] = v
	}
	return nil
}
