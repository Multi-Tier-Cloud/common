package p2pnode

import (
    "context"
    "errors"
    "fmt"
    "math"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/host"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p-core/protocol"
    "github.com/libp2p/go-libp2p-discovery"

    "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/multiformats/go-multiaddr"
)


// TODO: Move this to bootstrap at some point
var DefaultBootstrapPeers = []string{
    "/ip4/10.11.17.15/tcp/4001/ipfs/QmeZvvPZgrpgSLFyTYwCUEbyK6Ks8Cjm2GGrP2PA78zjAk",
    "/ip4/10.11.17.32/tcp/4001/ipfs/12D3KooWGegi4bWDPw9f6x2mZ6zxtsjR8w4ax1tEMDKCNqdYBt7X",
}

// Config is a structure for passing arguments
// into Node constructor NewNode
type Config struct {
    ListenAddrs        []string
    BootstrapPeers     []string
    StreamHandlers     []network.StreamHandler
    HandlerProtocolIDs []protocol.ID
    Rendezvous         []string
}

// Config constructor that returns default configuration
func NewConfig() Config {
    var config Config

    config.ListenAddrs        = []string{}
    config.BootstrapPeers     = DefaultBootstrapPeers
    config.StreamHandlers     = []network.StreamHandler{}
    config.HandlerProtocolIDs = []protocol.ID{}
    config.Rendezvous         = []string{}

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

// Node constructor
func NewNode(ctx context.Context, config Config) (Node, error) {
    var err error

    // Populate new node instance
    var node Node

    node.Ctx = ctx

    // Create a libp2p Host instance
    if len(config.ListenAddrs) != 0 {
        fmt.Println("Creating Libp2p host")
        listenAddrs, err := StringsToMultiaddrs(config.ListenAddrs)
        if err != nil {
            return node, err
        }
        node.Host, err = libp2p.New(node.Ctx,
            libp2p.ListenAddrs(listenAddrs...),
        )
        if err != nil {
            return node, err
        }
    } else {
        node.Host, err = libp2p.New(node.Ctx)
        if err != nil {
            return node, err
        }
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
        // Parse Bootstrap addresses
        bootstrapPeers, err := StringsToMultiaddrs(config.BootstrapPeers)
        if err != nil {
            return node, err
        }

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
            for _, peerAddr := range bootstrapPeers {
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
