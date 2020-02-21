package p2putil

import (
    "errors"
    "context"
    "fmt"
    "sort"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/p2p/protocol/ping"
    "github.com/libp2p/go-libp2p-core/host"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p-core/protocol"
    "github.com/libp2p/go-libp2p-discovery"

    "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/multiformats/go-multiaddr"
)

func StringsToMultiaddrs(stringMultiaddrs []string) ([]multiaddr.Multiaddr, error) {
    multiaddrs := make([]multiaddr.Multiaddr, 0)

    for _, s := range stringMultiaddrs {
        ma, err := multiaddr.NewMultiaddr(s)
        if err != nil {
            return multiaddrs, err
        }
        multiaddrs = append(multiaddrs, ma)
    }

    return multiaddrs, nil
}


type PeerInfo struct {
    RTT   time.Duration
    ID    peer.ID
    Addrs []multiaddr.Multiaddr
}

func SortPeers(peerChan <-chan peer.AddrInfo, Node Node) []PeerInfo {
	var peers []PeerInfo

    for p := range peerChan {
        responseChan := ping.Ping(Node.Ctx, Node.Host, p.ID)
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
