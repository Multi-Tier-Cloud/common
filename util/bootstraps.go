package util

import (
	"flag"
	"fmt"

	"github.com/multiformats/go-multiaddr"
)

type bootstrapAddrs []multiaddr.Multiaddr

var (
	// Stores the bootstrap multiaddrs
	bootstraps bootstrapAddrs

	// Used to avoid re-defining 'bootstraps' if AddbootstrapAddrs() is
	// called multiple times. After the first call, it should simply
	// return the slice of bootstrap addresses.
	bootstrapsFlagLoaded = false
)

func (addrs *bootstrapAddrs) String() string {
	return fmt.Sprintf("%v", *addrs)
}

func (addrs *bootstrapAddrs) Set(val string) error {
	newAddr, err := multiaddr.NewMultiaddr(val)
	if err != nil {
		return err
	}

	for _, ma := range *addrs {
		if ma.String() == newAddr.String() {
			return nil // Skip append
		}
	}

	*addrs = append(*addrs, newAddr)
	return nil
}

// Returns address to a slice of strings that will store the bootstrap
// multiaddresses once flag.Parse() is called (prior to that, it will
// be an empty slice).
func AddBootstrapFlags() (*bootstrapAddrs, error) {
	if !bootstrapsFlagLoaded {
		flag.Var(&bootstraps, "bootstrap",
			"Multiaddress of a bootstrap node.\n"+
				"This flag can be specified multiple times.")

		bootstrapsFlagLoaded = true
	}

	return &bootstraps, nil
}
