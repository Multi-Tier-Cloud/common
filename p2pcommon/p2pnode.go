package p2pnode

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

    "github.com/Multi-Tier-Cloud/common/libp2p-common/p2putil"
)


var DefaultBootstrapPeers = []string{
    "/ip4/10.11.17.15/tcp/4001/ipfs/QmeZvvPZgrpgSLFyTYwCUEbyK6Ks8Cjm2GGrP2PA78zjAk",
    "/ip4/10.11.17.32/tcp/4001/ipfs/12D3KooWGegi4bWDPw9f6x2mZ6zxtsjR8w4ax1tEMDKCNqdYBt7X",
}

type Config struct {
    ListenAddrs       []string
    BootstrapPeers    []string
    StreamHandler     func(stream network.Stream)
    HandlerProtocolID string
    Rendezvous        string
}

func NewConfig() Config {
    var config Config

    config.ListenAddrs       = []string{}
    config.BootstrapPeers    = DefaultBootstrapPeers
    config.StreamHandler     = nil
    config.HandlerProtocolID = ""
    config.Rendezvous        = ""

    return config
}

type Node struct {
    Ctx                context.Context
    Host               host.Host
    DHT                *dht.IpfsDHT
	RoutingDiscovery   *discovery.RoutingDiscovery
}

func NewNode(ctx context.Context, config Config) (Node, error) {
    var err error

    // Populate gobal node variable
    var node Node

    node.Ctx = ctx

    if len(config.ListenAddrs) != 0 {
        fmt.Println("Creating Libp2p node")
        listenAddrs, err := p2putil.StringsToMultiaddrs(config.ListenAddrs)
        if err != nil {
            return node, err
        }
        node.Host, err = libp2p.New(node.Ctx,
            libp2p.ListenAddrStrings(listenAddrs...),
        )
        if err != nil {
            return node, err
        }
    }

    if streamHandler != nil {
        fmt.Println("Setting stream handler")
        handlerProtocolID = protocol.ID(config.HandlerProtocolID)
        node.Host.SetStreamHandler(handlerProtocolID, config.StreamHandler)
    }

    fmt.Println("Creating DHT")
    node.DHT, err = dht.New(node.Ctx, node.Host)
    if err != nil {
        return node, err
    }

    bootstrapPeers, err := p2putil.StringsToMultiaddrs(config.BootstrapPeers)
    if err != nil {
        return node, err
    }

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

    if err = node.DHT.Bootstrap(node.Ctx); err != nil {
        return node, err
    }

    fmt.Println("Creating Routing Discovery")
    node.RoutingDiscovery = discovery.NewRoutingDiscovery(node.DHT)
    if config.Rendezvous != "" {
        discovery.Advertise(node.Ctx, node.RoutingDiscovery, config.Rendezvous)
    }

    fmt.Println("Finished setting up libp2p Node with PID", node.Host.ID())
    return node, nil
}
