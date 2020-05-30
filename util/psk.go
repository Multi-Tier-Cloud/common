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
	"crypto/rand"
	"flag"
	"fmt"
	"golang.org/x/crypto/sha3"

	"github.com/libp2p/go-libp2p-core/pnet"
)

// Need to define custom type to implement flag's Value interface.
// Don't want to expose it outside the package as it'll be redundant and
// possibly reduce readability and comprehension of the underlying type.
type pskValue pnet.PSK

// libp2p's PSK definition is a slice of 32 bytes
const PSK_NUM_BYTES = 32

var (
	// Stores the bootstrap multiaddrs
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

func (pskVal *pskValue) String() string {
	return string(*pskVal)
}

func (pskVal *pskValue) Set(s string) error {
	var err error
	var pnetPsk pnet.PSK
	if s == "" {
		pnetPsk, err = CreateRandPSK()
		if err != nil {
			return fmt.Errorf("Unable to create random PSK\n%w", err)
		}
	} else {
		pnetPsk, err = CreatePSK(s)
		if err != nil {
			return fmt.Errorf("Unable to create PSK from \"%s\"\n%w", s, err)
		}
	}

	// Cast and return
	*pskVal = (pskValue)(pnetPsk)
	return nil
}

// Sets the "-psk" flag and returns a pointer to a pre-shared key
func AddPSKFlag() (*pnet.PSK, error) {
	if !pskFlagLoaded {
		flag.Var(&psk, "psk",
			"Passphrase used to create a pre-shared key (PSK) used amongst nodes "+
				"to form a private network. The passphrase provided here will not be "+
				"stored in memory. It is HIGHLY RECOMMENDED you use a passphrase you "+
				"can easily memorize, or write it down somewhere safe. If you forget "+
				"the passphrase, you will be unable to join new nodes/services to "+
				"the same network.")

		pskFlagLoaded = true
	}

	// Cast and return
	return (*pnet.PSK)(&psk), nil
}

// For enabling tests, ideally should not be used.
// This is needed to return a pointer to type pskValue, a hidden type.
// This enables tests for the Set() and String() functions above.
func GetPSKPointer() *pskValue {
	return &psk
}
