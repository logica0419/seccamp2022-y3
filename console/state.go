package console

import (
	"fmt"
	"strconv"

	"sc.y3/peer"
)

func State(name string) (string, error) {
	var reply peer.RequestStateReply
	err := sendRPC(name, "Worker.RequestState", peer.RequestStateArgs{}, &reply)
	if err != nil {
		return "", err
	}

	return reply.State.String(), nil
}

func Log(name string) (string, error) {
	var reply peer.RequestLogReply
	err := sendRPC(name, "Worker.RequestLog", peer.RequestLogArgs{}, &reply)
	if err != nil {
		return "", err
	}

	logs := "Operation Logs:"
	for i, v := range reply.Logs {
		logs += fmt.Sprintf("\nOperation%d: %s %d", i+1, v.Operator, v.Operand)
	}

	return logs, nil
}

func UpdateState(name string, operator string, value string) (string, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return "", err
	}

	var reply peer.UpdateStateReply
	err = sendRPC(name, "Worker.UpdateState", peer.UpdateStateArgs{Operator: operator, Operand: i}, &reply)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\nBefore: %s\nAfter : %s", reply.Before.String(), reply.After.String()), nil
}
