package console

import (
	"fmt"
	"net/rpc"
	"time"

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
	var client *rpc.Client
	c := make(chan error)

	go func() {
		addr, err := disp.GetAddr(peer)
		if err != nil {
			c <- err
			return
		}

		client, err = rpc.Dial("tcp", addr)
		if err != nil {
			c <- err
			return
		}

		c <- client.Call(method, args, reply)
	}()

	t := time.NewTimer(time.Second * 1)

	select {
	case err := <-c:
		return err
	case <-t.C:
		return fmt.Errorf("RPC call timed out")
	}
}
