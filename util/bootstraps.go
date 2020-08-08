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
package util

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/multiformats/go-multiaddr"
)

// Need to define custom type to implement flag's Value interface.
// Don't want to expose it outside the package as it'll be redundant and
// possibly reduce readability and comprehension of the underlying type.
type bootstrapAddrs []multiaddr.Multiaddr

const ENV_KEY_BOOTSTRAPS = "P2P_BOOTSTRAPS"

var (
	// Stores the bootstrap multiaddrs
	bootstraps bootstrapAddrs

	// Used to avoid re-defining 'bootstraps' if AddBootstrapFlags() is
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

// Don't like this... feels hacky to have a function just for tests in the main package
// This is needed to return a pointer to type bootstrapAddrs, a hidden type.
// This enables tests for the Set() and String() functions above.
func GetBootstrapPointer() *bootstrapAddrs {
	return &bootstraps
}

// Returns address to a slice of strings that will store the bootstrap
// multiaddresses once flag.Parse() is called (prior to that, it will
// be an empty slice).
func AddBootstrapFlags() (*[]multiaddr.Multiaddr, error) {
	if !bootstrapsFlagLoaded {
		flag.Var(&bootstraps, "bootstrap",
			"Multiaddress of a bootstrap node.\n"+
				"This flag can be specified multiple times.\n"+
				fmt.Sprintf("Alternatively, an environment variable named %s can\n"+
					"be set with a space-separated list of bootstrap multiaddresses.",
					ENV_KEY_BOOTSTRAPS))

		bootstrapsFlagLoaded = true
	}

	// Cast and return
	return (*[]multiaddr.Multiaddr)(&bootstraps), nil
}

// If the environment variable does not exist, or if there are errors during
// parsing, return the 0-value of the return type.
func GetEnvBootstraps() ([]multiaddr.Multiaddr, error) {
	envStr := os.Getenv(ENV_KEY_BOOTSTRAPS)
	if envStr == "" {
		return nil, nil
	}

	bootstraps, err := StringsToMultiaddrs(strings.Fields(envStr))
	if err != nil {
		err = fmt.Errorf("ERROR: Unable to parse environment variable %s.\n%w",
			ENV_KEY_BOOTSTRAPS, err)

		return nil, err
	}

	return bootstraps, nil
}
