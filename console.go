package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"strings"

	"sc.y3/dispatcher"
	"sc.y3/peer"
	//"github.com/rivo/tview"
)

var (
	disp *dispatcher.Client
)

func main() {
	dispatcherFlag := flag.String("dispatcher", "localhost:8080", "Dispatcher address")
	flag.Parse()

	if *dispatcherFlag == "localhost:8080" && os.Getenv("DISPATCHER") != "" {
		*dispatcherFlag = os.Getenv("DISPATCHER")
	}

	var err error
	disp, err = dispatcher.FindDispatcher(*dispatcherFlag)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("? ")
		scanner.Scan()
		input := scanner.Text()
		command := parse(input)
		result, err := command.Exec()
		if err != nil {
			fmt.Printf("> [ERROR] %s\n", err)
		} else {
			fmt.Println("> ", result)
		}
	}
}

func parse(raw string) Command {
	ret := Command{"", make([]string, 0)}
	for i, v := range strings.Split(raw, " ") {
		switch i {
		case 0:
			ret.operation = v
		default:
			ret.args = append(ret.args, v)
		}
	}
	return ret
}

func send_rpc(peer, method string, args any, reply any) error {
	addr, err := disp.GetAddr(peer)
	if err != nil {
		return err
	}
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}

	return client.Call(method, args, reply)
}

type Command struct {
	operation string
	args      []string
}

func (c *Command) Exec() (string, error) {
	switch c.operation {
	case "state":
		return State(c.args[0])
	case "list":
		return ListPeers(c.args[0])
	case "log":
		return Log(c.args[0])
	case "update":
		return UpdateState(c.args[0], c.args[1], c.args[2])
	default:
		return "", fmt.Errorf("No such command")
	}
}

func State(name string) (string, error) {
	var reply peer.RequestStateReply
	err := send_rpc(name, "Worker.RequestState", peer.RequestStateArgs{}, &reply)
	if err != nil {
		return "", err
	}

	return reply.State.String(), nil
}

func ListPeers(name string) (string, error) {
	var reply peer.RequestConnectedPeersReply
	err := send_rpc(name, "Worker.RequestConnectedPeers", peer.RequestConnectedPeersArgs{}, &reply)
	if err != nil {
		return "", err
	}

	peers := fmt.Sprintf("Peers Connected to %s:", name)
	for k, v := range reply.Peers {
		peers += fmt.Sprintf("\n%s: %s", k, v)
	}

	return peers, nil
}

func Log(name string) (string, error) {
	var reply peer.RequestLogReply
	err := send_rpc(name, "Worker.RequestLog", peer.RequestLogArgs{}, &reply)
	if err != nil {
		return "", err
	}

	logs := "Operation Logs:"
	for i, v := range reply.Logs {
		logs += fmt.Sprintf("\nOperation%d: %s %d", i+1, v.Operation, v.Value)
	}

	return logs, nil
}

func UpdateState(name string, operation string, value string) (string, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return "", err
	}

	var reply peer.UpdateStateReply
	err = send_rpc(name, "Worker.UpdateState", peer.UpdateStateArgs{Operation: operation, Value: i}, &reply)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\nBefore: %s\nAfter : %s", reply.Before.String(), reply.After.String()), nil
}
