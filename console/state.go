package console

import (
	"fmt"
	"strconv"

	"sc.y3/peer"
)

func State(name string) (string, error) {
	var reply peer.RequestStateReply
	err := send_rpc(name, "Worker.RequestState", peer.RequestStateArgs{}, &reply)
	if err != nil {
		return "", err
	}

	return reply.State.String(), nil
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
