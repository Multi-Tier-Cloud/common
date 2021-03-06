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
	"crypto/rand"
	"flag"
	"fmt"
	"golang.org/x/crypto/sha3"
	"os"

	"github.com/libp2p/go-libp2p-core/pnet"
)

// Need to define custom type to implement flag's Value interface.
// Don't want to expose it outside the package as it'll be redundant and
// possibly reduce readability and comprehension of the underlying type.
type pskValue struct {
	hPsk pnet.PSK // Hashed 32-byte PSK usable by libp2p
	sPsk string   // Original un-hashed passphrase
}

const (
	// libp2p's PSK definition is a slice of 32 bytes
	PSK_NUM_BYTES = 32

	ENV_KEY_PSK = "P2P_PSK"
)

var (
	// Stores the PSK (both hashed and original string format)
	psk pskValue

	// Used to avoid re-defining 'psk' if AddPSKFlag() is
	// called multiple times. After the first call, it should simply
	// return a pointer to the psk.
	pskFlagLoaded = false
)

// Generates a random PSK
func CreateRandPSK() (pnet.PSK, error) {
	randBytes := make([]byte, PSK_NUM_BYTES)
	if size, err := rand.Read(randBytes); size != PSK_NUM_BYTES || err != nil {
		return nil, fmt.Errorf("Unable to generate random bytes")
	}

	digest := sha3.Sum256(randBytes)

	// Convert [32]byte to []byte
	return digest[:], nil
}

// Uses SHA256 to hash an input string into a 256-bit value.
// This matches libp2p's current implementation for PSK, which
// seems to be hard-coded to be 32 Bytes (256 bits) long.
//
// If the input string is the zero-value (i.e. ""), then this function
// call is equivalent to calling CreateRandPSK(), which generates a
// random pre-shared key.
func CreatePSK(psk string) (pnet.PSK, error) {
	if psk == "" {
		return CreateRandPSK()
	}

	digest := sha3.Sum256([]byte(psk))

	// Convert [32]byte to []byte
	return digest[:], nil
}

// Returns the last string used when Set() was called
func (pskVal *pskValue) String() string {
	return pskVal.sPsk
}

// Note that the values of sPsk and pskVal will be altered on each call
func (pskVal *pskValue) Set(s string) error {
	var err error
	if s == "" {
		pskVal.hPsk, err = CreateRandPSK()
		if err != nil {
			return fmt.Errorf("Unable to create random PSK\n%w", err)
		}
	} else {
		pskVal.hPsk, err = CreatePSK(s)
		if err != nil {
			return fmt.Errorf("Unable to create PSK from \"%s\"\n%w", s, err)
		}
	}

	// Save the original string passphrase
	pskVal.sPsk = s

	return nil
}

// Sets the "-psk" flag and returns a pointer to a pre-shared key
func AddPSKFlag() (*pnet.PSK, error) {
	if !pskFlagLoaded {
		flag.Var(&psk, "psk",
			"Passphrase used to create a pre-shared key (PSK) used amongst nodes\n"+
				"to form a private network. It is HIGHLY RECOMMENDED you use a\n"+
				"passphrase you can easily memorize, or write it down somewhere safe.\n"+
				"If you forget the passphrase, you will be unable to join new nodes\n"+
				"and services to the same network.\n"+
				fmt.Sprintf("Alternatively, an environment variable named %s can\n"+
					"be set with the passphrase.", ENV_KEY_PSK))

		pskFlagLoaded = true
	}

	// Cast and return
	return &psk.hPsk, nil
}

// For enabling tests, ideally should not be used.
// This is needed to return a pointer to type pskValue, a hidden type.
// This enables tests for the Set() and String() functions above.
func GetPSKPointer() *pskValue {
	return &psk
}

// If the environment variable does not exist, or if there are errors during
// parsing, return the 0-value of the return type.
func GetEnvPSK() (pnet.PSK, error) {
	envStr := GetEnvPSKString()
	if envStr == "" {
		return nil, nil
	}

	pnetPsk, err := CreatePSK(envStr)
	if err != nil {
		err = fmt.Errorf("ERROR: Unable to parse environment variable %s.\n%w",
			ENV_KEY_PSK, err)

		return nil, err
	}

	return pnetPsk, nil
}

// Returns un-hashed PSK passphrase specified in environment variable
func GetEnvPSKString() string {
	return os.Getenv(ENV_KEY_PSK)
}

// Returns the un-hashed PSK passphrase of the PSK from the
// last time Get() was called.
func GetFlagPSKString() string {
	return psk.sPsk
}
