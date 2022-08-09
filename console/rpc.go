package console

import (
	"net/rpc"

	"sc.y3/dispatcher"
)

func DialDispatcher(addr string) error {
	var err error
	disp, err = dispatcher.FindDispatcher(addr)
	if err != nil {
		return err
	}

	return nil
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
