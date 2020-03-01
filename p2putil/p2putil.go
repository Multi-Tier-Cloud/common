package p2putil

import (
    "io/ioutil"
    "sort"
    "time"

    "github.com/libp2p/go-libp2p/p2p/protocol/ping"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"

    "github.com/Multi-Tier-Cloud/common/p2pnode"
)

// Performance indicator
type PerfInd struct {
    RTT time.Duration
    // TODO: Add more fields than RTT
}

// PeerInfo holds information relative peer performance and contact information
type PeerInfo struct {
    Perf PerfInd
    ID   peer.ID
}

// Compares whether l performance is better than r performance
func (l PerfInd) Compare(r PerfInd) bool {
    return l.RTT < r.RTT
}

// Alternative versoin of Compare
func Compare(l, r PerfInd) bool {
    return l.Compare(r)
}

// Get performance indicators and return sorted peers based on it
func SortPeers(peerChan <-chan peer.AddrInfo, node p2pnode.Node) []PeerInfo {
	var peers []PeerInfo

    for p := range peerChan {
        responseChan := ping.Ping(node.Ctx, node.Host, p.ID)
        result := <-responseChan
        if len(p.Addrs) == 0 || result.RTT == 0 {
            continue
        }
        peers = append(peers, PeerInfo{Perf: PerfInd{RTT: result.RTT}, ID: p.ID})
	}

    sort.Slice(peers, func(i, j int) bool {
        return Compare(peers[i].Perf, peers[j].Perf)
    })

    return peers
}

// Read from stream
func ReadMsg(stream network.Stream) (data []byte, err error) {
	data, err = ioutil.ReadAll(stream)
	if err != nil {
		stream.Reset()
		return []byte{}, err
	}

	return data, nil
}

// Write to stream
func WriteMsg(stream network.Stream, data []byte) (err error) {
	_, err = stream.Write(data)
	if err != nil {
		stream.Reset()
		return err
	}

    stream.Close()
	return nil
}
