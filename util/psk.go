package util

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/sha3"

	"github.com/libp2p/go-libp2p-core/pnet"
)

// Generates a random PSK
func CreateRandPSK() (pnet.PSK, error) {
	randBytes := make([]byte, 32)
	if size, err := rand.Read(randBytes); size != 32 || err != nil {
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
