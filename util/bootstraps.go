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
