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

package util_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"

	"github.com/PhysarumSM/common/util"
)

// Creates a temp file for testing purposes.
// User should take care to delete the file before the test ends.
// Returns the system path to the file and an error if it exists.
func createTempFile() (string, error) {
	tmpFile, err := ioutil.TempFile("/tmp", "tmp")
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	return tmpFile.Name(), nil
}

func TestGeneratePrivKey(test *testing.T) {
	testCases := []struct {
		name      string
		algo      string
		bits      int
		shouldErr bool
	}{
		// Negative test cases
		{"RSA-small-bits", "rsa", 1024, true},
		{"Wrong-algo", "rsaa", 2048, true},

		// Positive test cases
		{"RSA-basic", "rsa", 2048, false},
		{"ECDSA-basic", "ecdsa", 2048, false},
		{"Ed25519-basic", "Ed25519", 0, false},
		{"Secp256k1-basic", "Secp256k1", 0, false},
	}

	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			_, err := util.GeneratePrivKey(testCase.algo, testCase.bits)
			if testCase.shouldErr {
				if err == nil {
					test.Errorf("Passed case (%s); Expected it to fail.", testCase.name)
				} else {
					test.Log(err)
				}
			} else if !testCase.shouldErr && err != nil {
				test.Log(err)
				test.Errorf("Failed case (%s); Expected it to pass.\n%v", testCase.name, err)
			}
		})
	}
}

func TestStoreKey(test *testing.T) {
	// Setup for case of existing key file
	existingFile, err := createTempFile()
	if err != nil {
		panic(err)
	}

	testCases := []struct {
		name      string
		algo      int
		bits      int
		shouldErr bool
	}{
		// Negative test case
		{"ExistingFile", crypto.RSA, 2048, true},

		// Positive test cases
		{"RSA", crypto.RSA, 2048, false},
		{"Ed25519", crypto.Ed25519, 0, false},
		{"Secp256k1", crypto.Secp256k1, 0, false},
		{"ECDSA", crypto.ECDSA, 0, false},
		//{"TildeExpansion", "RSA", 2048, false},
	}

	var tmpFile string
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			// Create a dummy private key to be used for tests
			// Assume libp2p's crypto was properly tested by their devs
			priv, _, err := crypto.GenerateKeyPair(testCase.algo, testCase.bits)
			if err != nil {
				test.Fatalf("Unable to generate test key: libp2p's crypto pkg returned an error")
			}

			// This is a shitty hack... breaks generalization
			if testCase.name == "ExistingFile" {
				tmpFile = existingFile
			} else {
				tmpFile = "/tmp/tmp" + string(rand.Int())
			}

			err = util.StorePrivKeyToFile(priv, tmpFile)
			if testCase.shouldErr {
				if err == nil {
					test.Errorf("Passed case (%s); Expected it to fail.", testCase.name)
				} else {
					test.Log(err)
				}
			} else if !testCase.shouldErr && err != nil {
				test.Log(err)
				test.Errorf("Failed case (%s); Expected it to pass.\n%v", testCase.name, err)
			}

			if !testCase.shouldErr {
				// Check that the key exists
				_, err = os.Stat(tmpFile)
				if os.IsNotExist(err) {
					test.Errorf("Expected key file (%s) does not exist.", tmpFile)
				}
			}

			os.Remove(tmpFile)
		})
	}
}

func TestLoadKey(test *testing.T) {
	// Create an existing key to load from
	keyType := pb.KeyType(3)
	keyB64 := "MHcCAQEEIHp/bhcT3Jge9ykOMjk+AgCi6qqM8it01IRoRbXphHXaoAoGCCqGSM49AwEHoUQDQgAEhN7JYn9DN9POlfbkDwR1T74gxPpUx90cWxbuyuvOL10DsQe1UD/IVBxdQ1nZPaYC/m+nSaUdZ53gFBaHLQg+QQ=="

	tmpFile, err := ioutil.TempFile("/tmp", "tmp")
	if err != nil {
		panic(err)
	}
	tmpFile.WriteString(fmt.Sprintf("%d %s\n", keyType, keyB64))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	priv, err := util.LoadPrivKeyFromFile(tmpFile.Name())
	if err != nil {
		test.Fatalf("loadPrivKeyFromFile() failed with error:\n%v", err)
	}

	if priv.Type() != keyType {
		test.Fatalf("Incorrect key type loaded (%d), was expecting %d", priv.Type(), keyType)
	}

	rawBytes, err := priv.Raw()
	if err != nil {
		test.Fatalf("Could not load raw bytes from loaded key")
	}

	loadedKeyB64 := crypto.ConfigEncodeKey(rawBytes)
	if loadedKeyB64 != keyB64 {
		test.Fatalf("Loaded key is not identical to test key.\n"+
			"Loaded: %s\nExpect: %s\n", loadedKeyB64, keyB64)
	}
}

func TestKeyFlags(test *testing.T) {
	// Get a random file name... ensure it actually doesn't exist
	tmpFile, err := createTempFile()
	if err != nil {
		panic(err)
	}
	os.Remove(tmpFile)

	// Test adding flags
	keyFlags, err := util.AddKeyFlags(tmpFile)
	if err != nil {
		test.Fatalf("AddKeyFlags() failed:\n%v", err)
	}

	if keyFlags.Algo == nil || keyFlags.Bits == nil ||
		keyFlags.Keyfile == nil || keyFlags.Ephemeral == nil {

		test.Fatalf("Returned KeyFlags structure contains nil pointers")
	}
}

func TestCreateOrLoadKey(test *testing.T) {
	keyFlags := util.KeyFlags{}

	test.Run("EmptyKeyFlags", func(test *testing.T) {
		_, err := util.CreateOrLoadKey(keyFlags)
		if err == nil {
			test.Fatalf("CreateOrLoadKey() passed with empty KeyFlags; Expected it to fail.")
		}
	})

	var algo string
	var bits int
	var keyfile string
	var ephemeral bool

	keyFlags.Algo = &algo
	keyFlags.Bits = &bits
	keyFlags.Keyfile = &keyfile
	keyFlags.Ephemeral = &ephemeral

	test.Run("ZeroValFlags", func(test *testing.T) {
		_, err := util.CreateOrLoadKey(keyFlags)
		if err == nil {
			test.Fatalf("CreateOrLoadKey() passed with zero-valued KeyFlags variables; Expected it to fail.")
		}
	})

	algo = "rsa"
	bits = 2048
	keyfile, err := createTempFile() // Get random path for non-existent file
	if err != nil {
		panic(err)
	}
	os.Remove(keyfile)

	test.Run("ProperKeyFlags", func(test *testing.T) {
		priv, err := util.CreateOrLoadKey(keyFlags)
		if err != nil {
			test.Logf("CreateOrLoadKey() failed; Expected it to pass.\n%v", err)
			test.Logf("\tkeyFlags.Algo = %s\n", algo)
			test.Logf("\tkeyFlags.Bits = %d\n", bits)
			test.Logf("\tkeyFlags.Keyfile = %s\n", keyfile)
			test.Logf("\tkeyFlags.Ephemeral = %t\n", ephemeral)
			test.FailNow()
		}

		if priv == nil {
			test.Errorf("CreateOrLoadKey() returned a nil private key")
		}
	})
}
