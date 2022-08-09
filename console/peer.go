package console

import (
	"fmt"

	"sc.y3/peer"
)

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
