package console

import (
	"fmt"

	"sc.y3/peer"
)

func ListPeers(name string) (string, error) {
	var reply peer.RequestConnectedPeersReply
	err := sendRPC(name, "Worker.RequestConnectedPeers", peer.RequestConnectedPeersArgs{}, &reply)
	if err != nil {
		return "", err
	}

	peers := fmt.Sprintf("Peers Connected to %s:", name)
	for k, v := range reply.Peers {
		peers += fmt.Sprintf("\n%s: %s", k, v)
	}

	return peers, nil
}

func Leader(name string) (string, error) {
	var reply peer.RequestLeaderReply
	err := sendRPC(name, "Worker.RequestLeader", peer.RequestLeaderArgs{}, &reply)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Leader: %s", reply.Leader), nil
}
