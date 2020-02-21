package p2putil

import (
    "io/ioutil"
    "sort"
    "time"

    "github.com/libp2p/go-libp2p/p2p/protocol/ping"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"

    "github.com/multiformats/go-multiaddr"

    "github.com/Multi-Tier-Cloud/common/p2pnode"
)

type PeerInfo struct {
    RTT   time.Duration
    ID    peer.ID
    Addrs []multiaddr.Multiaddr
}

func SortPeers(peerChan <-chan peer.AddrInfo, node p2pnode.Node) []PeerInfo {
	var peers []PeerInfo

    for p := range peerChan {
        responseChan := ping.Ping(node.Ctx, node.Host, p.ID)
        result := <-responseChan
        if len(p.Addrs) == 0 || result.RTT == 0 {
            continue
        }
        peers = append(peers, PeerInfo{RTT: result.RTT, ID: p.ID, Addrs: p.Addrs})
	}

    sort.Slice(peers, func(i, j int) bool {
        return peers[i].RTT < peers[j].RTT
    })

    return peers
}


func ReadMsg(stream network.Stream) (data []byte, err error) {
	data, err = ioutil.ReadAll(stream)
	if err != nil {
		stream.Reset()
		return []byte{}, err
	}

	return data, nil
}

func WriteMsg(stream network.Stream, data []byte) (err error) {
	_, err = stream.Write(data)
	if err != nil {
		stream.Reset()
		return err
	}

    stream.Close()
	return nil
}
