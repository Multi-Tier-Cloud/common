/* Copyright 2020 PhysarumSM Development Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package p2putil

import (
    "context"
    "io/ioutil"
    "sort"
    "time"

    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p/p2p/protocol/ping"

    "github.com/PhysarumSM/common/p2pnode"
)

// Performance indicator
type PerfInd struct {
    RTT time.Duration
    // TODO: Add more fields than RTT
}

// PeerInfo holds information relative peer performance and contact information
type PeerInfo struct {
    ID          peer.ID
    Perf        PerfInd
    ServName    string
    ServHash    string
}

// Compares whether l performance is less than r performance
// TODO: Figure out how to handle comparison if PerfInd contains more
//       than a single metric
func (l PerfInd) LessThan(r PerfInd) bool {
    return l.RTT < r.RTT
}

func (l PerfInd) GreaterThan(r PerfInd) bool {
    return l.RTT > r.RTT
}

func (l PerfInd) Equal(r PerfInd) bool {
    return l.RTT == r.RTT
}

// Get performance indicators and return sorted peers based on it
func SortPeers(peerChan <-chan peer.AddrInfo, node p2pnode.Node) []PeerInfo {
    var peers []PeerInfo

    // Set context with 1 second timeout for ping results for *all* peers.
    //
    // TODO: Move towards long-term solution to query a database for peer
    //       latency info, or some type of cache-like datastructure that's
    //       automatically updated, so we don't have to explicitly ping.
    ctx, cancel := context.WithTimeout(node.Ctx, time.Second)
    for p := range peerChan {
        responseChan := ping.Ping(ctx, node.Host, p.ID)
        result := <-responseChan
        if len(p.Addrs) == 0 || result.RTT == 0 {
            continue
        }
        peers = append(peers, PeerInfo{Perf: PerfInd{RTT: result.RTT}, ID: p.ID})
    }
    cancel()

    sort.Slice(peers, func(i, j int) bool {
        return peers[i].Perf.LessThan(peers[j].Perf)
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
