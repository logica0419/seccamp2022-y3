package peer

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type WorkerState struct {
	Value  int
	Leader string
	Term   int
}

func (s *WorkerState) String() (state string) {
	return fmt.Sprintf("Value: %d | Leader: %s, Term: %d", s.Value, s.Leader, s.Term)
}

type LogState struct {
	CommitIndex  int
	NextIndices  map[string]int
	MatchIndices map[string]int
}

func (s *LogState) String() (state string) {
	state += fmt.Sprintf("CommitIndex: %d", s.CommitIndex)

	state += "\nNextIndices"
	for k, v := range s.NextIndices {
		state += fmt.Sprintf("\n  [%s]: %d", k, v)
	}

	state += "\nMatchIndices"
	for k, v := range s.MatchIndices {
		state += fmt.Sprintf("\n  [%s]: %d", k, v)
	}

	return
}

type WorkerLog struct {
	Index    int
	Operator string
	Operand  int
}

type WorkerLogs []*WorkerLog

func (l WorkerLogs) find(index int) *WorkerLog {
	for _, v := range l {
		if v.Index == index {
			return v
		}
	}

	return nil
}

func (l WorkerLogs) sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Index < l[j].Index
	})
}

type Worker struct {
	name string
	node *Node

	mu sync.Mutex

	logs         WorkerLogs
	commitIndex  int
	nextIndices  map[string]int
	matchIndices map[string]int
	imu          sync.RWMutex

	term   int
	leader string

	heartBeatDuration time.Duration
	heartBeatTicker   *time.Ticker
	voteDuration      time.Duration
	voteTimer         *time.Timer
}

type WorkerOption func(*Worker)

func NewWorker(name string) *Worker {
	w := new(Worker)
	w.name = name
	w.logs = WorkerLogs{}
	w.commitIndex = -1
	w.nextIndices = make(map[string]int)
	w.matchIndices = make(map[string]int)
	w.term = 0

	w.heartBeatDuration = 1 * time.Second
	w.heartBeatTicker = time.NewTicker(w.heartBeatDuration)
	w.heartBeatTicker.Stop()

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

	w.voteDuration = time.Duration(w.Rand().ExpFloat64()*1000)*time.Millisecond + time.Second*3/2
	w.voteTimer = time.NewTimer(w.voteDuration)
	go w.StartVoteTimer()
}

func (w *Worker) Term() int {
	return w.term
}

func (w *Worker) SetTerm(term int) {
	w.term = term
}

func (w *Worker) Leader() string {
	return w.leader
}

func (w *Worker) SetLeader(leader string) {
	w.leader = leader
}

func (w *Worker) Logs() WorkerLogs {
	return w.logs
}

func (w *Worker) AddLog(l *WorkerLog) error {
	if l.Operator != "+" && l.Operator != "-" && l.Operator != "*" && l.Operator != "/" {
		return fmt.Errorf("invalid operator: %s", l.Operator)
	}

	w.logs = append(w.logs, l)
	return nil
}

func (w *Worker) DeleteLastLog() error {
	if len(w.logs) == 0 {
		return fmt.Errorf("no logs to delete")
	}

	w.logs = w.logs[:len(w.logs)-1]
	return nil
}

func (w *Worker) WorkerState() WorkerState {
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

	return WorkerState{
		Value:  temp,
		Leader: w.leader,
		Term:   w.term,
	}
}

func (w *Worker) LogState() LogState {
	return LogState{
		CommitIndex:  w.commitIndex,
		NextIndices:  w.nextIndices,
		MatchIndices: w.matchIndices,
	}
}

var ErrNotLeader = fmt.Errorf("not leader")

func (w *Worker) SendHeartBeat() error {
	index := -1
	if len(w.logs) > 0 {
		index = w.logs[len(w.logs)-1].Index
	}

	eg := errgroup.Group{}

	for k := range w.ConnectedPeers() {
		k := k

		eg.Go(func() error {
			for {
				reply := HeartBeatReply{}

				entry := WorkerLogs{}
				w.imu.RLock()
				for _, v := range w.logs {
					if v.Index >= w.nextIndices[k] {
						entry = append(entry, v)
					}
				}

				err := w.RemoteCallWithTimeout(k, "Worker.HeartBeat", HeartBeatArgs{
					Leader:       w.Name(),
					Term:         w.Term(),
					Entry:        entry,
					PrevLogIndex: w.nextIndices[k] - 1,
					LeaderCommit: w.commitIndex,
				}, &reply, w.heartBeatDuration/2)
				if err != nil {
					return err
				}
				w.imu.RUnlock()

				w.imu.Lock()
				defer w.imu.Unlock()

				if reply.Updated {
					w.nextIndices[k] = index + 1
					w.matchIndices[k] = index
					return nil
				}

				w.nextIndices[k]--
			}
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	for i := w.commitIndex + 1; i <= index; {
		committed := 0
		for _, v := range w.matchIndices {
			if v >= i {
				committed++
			}
		}

		if committed > len(w.ConnectedPeers())/2 || len(w.ConnectedPeers()) == 0 {
			w.commitIndex = i
			i++
		} else {
			break
		}
	}

	return nil
}

func (w *Worker) StartHeartBeatTicker() {
	w.ResetHeartBeatTicker()

	for {
		<-w.heartBeatTicker.C
		if w.Leader() != w.Name() {
			break
		}

		err := w.SendHeartBeat()

		if errors.Is(err, ErrNotLeader) {
			break
		}
	}

	w.heartBeatTicker.Stop()
	go w.StartVoteTimer()
}

func (w *Worker) ResetHeartBeatTicker() {
	w.heartBeatTicker.Reset(w.heartBeatDuration)
}

func (w *Worker) StartVoteTimer() {
	w.ResetVoteTimer()
	<-w.voteTimer.C

	log.Print("sending vote")

	w.LockMutex()
	w.SetTerm(w.Term() + 1)
	w.UnlockMutex()

	eg := errgroup.Group{}
	ac := make(chan bool, len(w.ConnectedPeers()))
	for k := range w.ConnectedPeers() {
		k := k

		eg.Go(func() error {
			reply := VoteReply{}
			err := w.RemoteCallWithTimeout(k, "Worker.Vote", VoteArgs{w.Name(), w.Term()}, &reply, w.heartBeatDuration/2)
			if err != nil {
				return err
			}

			ac <- reply.IfAccept

			return nil
		})
	}

	_ = eg.Wait()
	close(ac)

	acceptance := 0
	rejection := 0
	for a := range ac {
		if a {
			acceptance++
		} else {
			rejection++
		}
	}

	if acceptance > rejection || acceptance+rejection == 0 {
		log.Print("vote accepted")

		w.LockMutex()
		w.SetLeader(w.Name())
		w.UnlockMutex()

		go w.StartHeartBeatTicker()
		return
	}

	log.Print("vote rejected")
	go w.StartVoteTimer()
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
	err := w.node.call(name, method, args, reply)
	if err != nil {
		if errors.Is(err, rpc.ErrShutdown) {
			_ = w.node.Disconnect(name)
		}

		return err
	}

	return nil
}

func (w *Worker) RemoteCallWithTimeout(name, method string, args any, reply any, timeout time.Duration) error {
	c := make(chan error, 1)
	t := time.NewTimer(timeout)

	go func() {
		c <- w.node.call(name, method, args, reply)
	}()

	select {
	case err := <-c:
		if errors.Is(err, rpc.ErrShutdown) {
			_ = w.node.Disconnect(name)
		}
		return err

	case <-t.C:
		_ = w.node.Disconnect(name)
		return fmt.Errorf("call timed out")
	}
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
