package p2pnode

import (
    "context"
    "errors"
    "fmt"
    "sync"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/host"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    "github.com/libp2p/go-libp2p-core/protocol"
    "github.com/libp2p/go-libp2p-discovery"

    "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/multiformats/go-multiaddr"
)


var DefaultBootstrapPeers = []string{
    "/ip4/10.11.17.15/tcp/4001/ipfs/QmeZvvPZgrpgSLFyTYwCUEbyK6Ks8Cjm2GGrP2PA78zjAk",
    "/ip4/10.11.17.32/tcp/4001/ipfs/12D3KooWGegi4bWDPw9f6x2mZ6zxtsjR8w4ax1tEMDKCNqdYBt7X",
}

type Config struct {
    ListenAddrs        []string
    BootstrapPeers     []string
    StreamHandlers     []network.StreamHandler
    HandlerProtocolIDs []protocol.ID
    Rendezvous         []string
}

func NewConfig() Config {
    var config Config

    config.ListenAddrs        = []string{}
    config.BootstrapPeers     = DefaultBootstrapPeers
    config.StreamHandlers     = []network.StreamHandler{}
    config.HandlerProtocolIDs = []protocol.ID{}
    config.Rendezvous         = []string{}

    return config
}

type Node struct {
    Ctx                context.Context
    Host               host.Host
    DHT                *dht.IpfsDHT
    RoutingDiscovery   *discovery.RoutingDiscovery
}

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

func NewNode(ctx context.Context, config Config) (Node, error) {
    var err error

    // Populate gobal node variable
    var node Node

    node.Ctx = ctx

    if len(config.ListenAddrs) != 0 {
        fmt.Println("Creating Libp2p node")
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

    fmt.Println("Creating DHT")
    node.DHT, err = dht.New(node.Ctx, node.Host)
    if err != nil {
        return node, err
    }

    bootstrapPeers, err := StringsToMultiaddrs(config.BootstrapPeers)
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
    for _, rendezvous := range config.Rendezvous {
        if rendezvous != "" {
            discovery.Advertise(node.Ctx, node.RoutingDiscovery, rendezvous)
        } else {
            return node, errors.New("Cannot have empty Rendezvous element")
        }
    }

    fmt.Println("Finished setting up libp2p Node with PID", node.Host.ID())
    return node, nil
}
