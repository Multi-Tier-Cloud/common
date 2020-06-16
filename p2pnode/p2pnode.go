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
    "log"
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

    "github.com/Multi-Tier-Cloud/common/util"
)

func init() {
    log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
}

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
    Close              context.CancelFunc
    Host               host.Host
    DHT                *dht.IpfsDHT
    RoutingDiscovery   *discovery.RoutingDiscovery
    NetworkCallbacks   *network.NotifyBundle
}

const (
    MaxConnAttempts = 5

    // 512 seconds = 8 mins 32 secs
    MaxBackoffSecs = 512
)

func (node *Node) Advertise(rendezvous string) error {
    if rendezvous == "" {
        log.Printf("ERROR: Empty rendezvous string")
        return errors.New("Cannot have empty Rendezvous string")
    } else if node.RoutingDiscovery == nil {
        log.Printf("ERROR: RoutingDiscovery does not exist")
        return errors.New("No Discovery object available to advertise from")
    }

    discovery.Advertise(node.Ctx, node.RoutingDiscovery, rendezvous)

    return nil
}

// Returns a callback function for peer disconnection events
//
// Given the Node and the original Config used to create it, always try to
// maintain its connectivity to the original bootstraps (i.e. reconnect to
// them if they are disconnected. Upon reconnection, re-advertise any
// services and/or content.
func ReconnectCB(node *Node, cfg *Config) func(network.Network, network.Conn) {

    return func(net network.Network, conn network.Conn) {
        // If the context has been cancelled, we should not try to reconnect
        if node.Ctx.Err() != nil {
            return
        }

        var err error
        var addrInfo *peer.AddrInfo
        isBootstrap := false
        for _, peerAddr := range cfg.BootstrapPeers {
            addrInfo, err = peer.AddrInfoFromP2pAddr(peerAddr)
            if err != nil {
                log.Printf("ERROR: Unable to parse AddrInfo from %s\n%w\n", peerAddr, err)
                continue
            }

            if conn.RemotePeer() == addrInfo.ID {
                isBootstrap = true
                break
            }
        }

        if !isBootstrap {
            return
        }

        log.Printf("Connection to %s lost, attempting to reconnect...\n", conn.RemotePeer())

        // The disconnecting peer is a bootstrap, attempt reconnect
        // Perform exponential backoff until MaxBackoffSecs, then continue trying
        // forever once every MaxBackoffSecs until success.
        connAttempts := 0
        sleepDuration := 0

        for net.Connectedness(conn.RemotePeer()) != network.Connected {
            // Perform simple exponential backoff
            // TODO: Move this to helper function
            if connAttempts > 0 {
                sleepDuration = int(math.Pow(2, float64(connAttempts)))
                log.Printf("Reconnection to %s failed.\n", conn.RemotePeer())
				for i := 0; i < sleepDuration; i++ {
                    fmt.Printf("\rRetrying again in %d seconds...     ", sleepDuration - i)
                    time.Sleep(time.Second)

                    // Check if context has been cancelled, abort attempts
                    if node.Ctx.Err() != nil {
                        return
                    }
                }
                fmt.Println()
            }

            if err := node.Host.Connect(node.Ctx, *addrInfo); err != nil {
                log.Println(err)
            } else {
                log.Println("Reconnected to node:", addrInfo)
            }

            // connAttempts used to calculate sleep duration
            // Avoid incrementing when exceeding MaxBackoffSecs
            if sleepDuration < MaxBackoffSecs {
                connAttempts++
            }
        }

        // Re-advertise any rendezvous srings
        for _, r := range cfg.Rendezvous {
            node.Advertise(r)
        }
    }
}

// Node constructor
func NewNode(ctx context.Context, config Config) (Node, error) {
    var err error

    // Populate new node instance
    var node Node

    node.Ctx, node.Close = context.WithCancel(ctx)
    nodeOpts := []libp2p.Option{}

    // Set private key (for identity) if it exists
    if (config.PrivKey != nil) {
        nodeOpts = append(nodeOpts, libp2p.Identity(config.PrivKey))
    }

    // Set listen addresses if they exist
    if len(config.ListenAddrs) != 0 {
        listenAddrs, err := util.StringsToMultiaddrs(config.ListenAddrs)
        if err != nil {
            return node, err
        }

        nodeOpts = append(nodeOpts, libp2p.ListenAddrs(listenAddrs...))
    }

    // Set pre-sharked key (for private network) if it exists
    if (config.PSK != nil) {
        log.Println("Pre-shared key detected, node will belong to a private network")
        nodeOpts = append(nodeOpts, libp2p.PrivateNetwork(config.PSK))
    }

    // Create a libp2p Host instance
    log.Println("Creating new p2p host")
    node.Host, err = libp2p.New(node.Ctx, nodeOpts...)
    if err != nil {
        return node, err
    }

    // Register Stream Handlers and corresponding Protocol IDs
    if len(config.HandlerProtocolIDs) != len(config.StreamHandlers) {
        return node, errors.New("StreamHandlers and HandlerProtocolIDs must map one-to-one")
    }
    log.Println("Setting stream handlers")
    for i := range config.HandlerProtocolIDs {
        if config.HandlerProtocolIDs[i] != "" && config.StreamHandlers[i] != nil {
            node.Host.SetStreamHandler(config.HandlerProtocolIDs[i], config.StreamHandlers[i])
        } else {
            return node, errors.New("Cannot have empty StreamHandler/HandlerProtocolID element")
        }
    }

    // Create a libp2p DHT instance
    log.Println("Creating DHT")
    node.DHT, err = dht.New(node.Ctx, node.Host, dht.Mode(dht.ModeServer))
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

            log.Println("Connecting to bootstrap nodes...")
            var wg sync.WaitGroup
            for _, peerAddr := range config.BootstrapPeers {
                peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
                if err != nil {
                    return node, fmt.Errorf("ERROR: Unable to parse AddrInfo from %s\n%w\n", peerAddr, err)
                }

                wg.Add(1)
                go func(addr peer.AddrInfo) {
                    defer wg.Done()
                    if err := node.Host.Connect(node.Ctx, addr); err != nil {
                        log.Println(err)
                    } else {
                        log.Println("Connected to bootstrap node:", addr)
                    }
                }(*peerinfo)
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

        log.Println("Connected to", numConnected, "peers!")
    } else {
        log.Println("No bootstraps provided, not connecting to any peers")
    }

    if err = node.DHT.Bootstrap(node.Ctx); err != nil {
        return node, err
    }

    // Create and register network callbacks. Use a disconnection notifier
    // to monitor when bootstraps disconnect, and attempt to reconnect.
    // Users can override or add any other callbacks they want, either
    // directly to the NotifyBundle created here, or register their own.
    netCBs := network.NotifyBundle{}
    netCBs.DisconnectedF = ReconnectCB(&node, &config)
    node.NetworkCallbacks = &netCBs
    node.Host.Network().Notify(node.NetworkCallbacks)

    // Create a libp2p Routing Discovery instance
    log.Println("Creating Routing Discovery")
    node.RoutingDiscovery = discovery.NewRoutingDiscovery(node.DHT)
    for _, rendezvous := range config.Rendezvous {
        if rendezvous != "" {
            discovery.Advertise(node.Ctx, node.RoutingDiscovery, rendezvous)
        } else {
            return node, errors.New("Cannot have empty Rendezvous element")
        }
    }

    // node initialization finished
    log.Println("Finished setting up libp2p Node with PID", node.Host.ID(),
                "and Multiaddresses", node.Host.Addrs())
    return node, nil
}
