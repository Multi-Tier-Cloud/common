/* Copyright 2020 Multi-Tier-Cloud Development Team
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
package p2pnode

import (
    "context"
    "errors"
    "fmt"
    "math"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/crypto"
    "github.com/libp2p/go-libp2p-core/host"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p-core/pnet"
    "github.com/libp2p/go-libp2p-core/protocol"
    "github.com/libp2p/go-libp2p-discovery"

    "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/multiformats/go-multiaddr"
)


// Config is a structure for passing arguments
// into Node constructor NewNode
type Config struct {
    PrivKey            crypto.PrivKey
    ListenAddrs        []string
    BootstrapPeers     []multiaddr.Multiaddr
    StreamHandlers     []network.StreamHandler
    HandlerProtocolIDs []protocol.ID
    Rendezvous         []string
    PSK                pnet.PSK
}

// Config constructor that returns default configuration
func NewConfig() Config {
    var config Config
    return config
}

// Node is a struct that holds all libp2p related objects
// for a node instance
type Node struct {
    Ctx                context.Context
    Host               host.Host
    DHT                *dht.IpfsDHT
    RoutingDiscovery   *discovery.RoutingDiscovery
}

// Helper function to cast a slice of strings into a slice of Multiaddrs
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

const (
    MaxConnAttempts = 5
)

func (node *Node) Advertise(rendezvous string) error {
    if rendezvous != "" {
        discovery.Advertise(node.Ctx, node.RoutingDiscovery, rendezvous)
    } else {
        return errors.New("Cannot have empty Rendezvous string")
    }
    return nil
}

// Node constructor
func NewNode(ctx context.Context, config Config) (Node, error) {
    var err error

    // Populate new node instance
    var node Node

    node.Ctx = ctx
    nodeOpts := []libp2p.Option{}

    // Set private key (for identity) if it exists
    if (config.PrivKey != nil) {
        nodeOpts = append(nodeOpts, libp2p.Identity(config.PrivKey))
    }

    // Set listen addresses if they exist
    if len(config.ListenAddrs) != 0 {
        listenAddrs, err := StringsToMultiaddrs(config.ListenAddrs)
        if err != nil {
            return node, err
        }

        nodeOpts = append(nodeOpts, libp2p.ListenAddrs(listenAddrs...))
    }

    // Set pre-sharked key (for private network) if it exists
    if (config.PSK != nil) {
        fmt.Println("Pre-shared key detected, node will belong to a private network")
        nodeOpts = append(nodeOpts, libp2p.PrivateNetwork(config.PSK))
    }

    // Create a libp2p Host instance
    fmt.Println("Creating new p2p host")
    node.Host, err = libp2p.New(node.Ctx, nodeOpts...)
    if err != nil {
        return node, err
    }

    // Register Stream Handlers and corresponding Protocol IDs
    if len(config.HandlerProtocolIDs) != len(config.StreamHandlers) {
        return node, errors.New("StreamHandlers and HandlerProtocolIDs must map one-to-one")
    }
    fmt.Println("Setting stream handlers")
    for i := range config.HandlerProtocolIDs {
        if config.HandlerProtocolIDs[i] != "" && config.StreamHandlers[i] != nil {
            node.Host.SetStreamHandler(config.HandlerProtocolIDs[i], config.StreamHandlers[i])
        } else {
            return node, errors.New("Cannot have empty StreamHandler/HandlerProtocolID element")
        }
    }

    // Create a libp2p DHT instance
    fmt.Println("Creating DHT")
    node.DHT, err = dht.New(node.Ctx, node.Host)
    if err != nil {
        return node, err
    }

    // If bootstraps provided, ensure at least 1 must connect
    // If none provided, no intention to connect to bootstraps, so move on
    if len(config.BootstrapPeers) > 0 {
        numConnected := 0
        bootstrapAttempts := 0

        // Connect to bootstrap nodes
        // Perform exponential backoff until at least one successful connection,
        // is made, up to MaxConnAttempts attempts
        for numConnected == 0 && bootstrapAttempts < MaxConnAttempts {
            // Perform simple exponential backoff
            // TODO: Move this to helper function
            if bootstrapAttempts > 0 {
                sleepDuration := int(math.Pow(2, float64(bootstrapAttempts)))
                for i := 0; i < sleepDuration; i++ {
                    fmt.Printf("\rUnable to connect to any peers, retrying in %d seconds...     ", sleepDuration - i)
                    time.Sleep(time.Second)
                }
                fmt.Println()
            }

            bootstrapAttempts++

            fmt.Println("Connecting to bootstrap nodes...")
            var wg sync.WaitGroup
            for _, peerAddr := range config.BootstrapPeers {
                peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
                wg.Add(1)
                go func() {
                    defer wg.Done()
                    if err := node.Host.Connect(node.Ctx, *peerinfo); err != nil {
                        fmt.Println(err)
                    } else {
                        fmt.Println("Connected to bootstrap node:", *peerinfo)
                    }
                }()
            }
            wg.Wait()

            // Count only connections whose internal state is Connected
            for _, peerID := range node.Host.Network().Peers() {
                if node.Host.Network().Connectedness(peerID) == network.Connected {
                    numConnected++
                }
            }
        }

        if numConnected == 0 {
            return node, errors.New("Failed to connect to any bootstraps")
        }

        fmt.Println("Connected to", numConnected, "peers!")
    } else {
        fmt.Println("No bootstraps provided, not connecting to any peers")
    }

    if err = node.DHT.Bootstrap(node.Ctx); err != nil {
        return node, err
    }

    // Create a libp2p Routing Discovery instance
    fmt.Println("Creating Routing Discovery")
    node.RoutingDiscovery = discovery.NewRoutingDiscovery(node.DHT)
    for _, rendezvous := range config.Rendezvous {
        if rendezvous != "" {
            discovery.Advertise(node.Ctx, node.RoutingDiscovery, rendezvous)
        } else {
            return node, errors.New("Cannot have empty Rendezvous element")
        }
    }

    // node initialization finished
    fmt.Println("Finished setting up libp2p Node with PID", node.Host.ID(),
                "and Multiaddresses", node.Host.Addrs())
    return node, nil
}
