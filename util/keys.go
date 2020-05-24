package util

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

const (
	RSA_MIN_BITS = 2048
)

func GeneratePrivKey(algo string, bits int) (crypto.PrivKey, error) {
	var keyType int
	for algoName, algoID := range pb.KeyType_value {
		if strings.EqualFold(algoName, algo) {
			keyType = int(algoID)
			break
		}
		keyType = -1
	}

	if keyType < 0 {
		return nil, fmt.Errorf("Unknown algorithm")
	} else if keyType == crypto.RSA && bits < RSA_MIN_BITS {
		return nil, fmt.Errorf("Number of bits for RSA must be at least %d", RSA_MIN_BITS)
	}

	// Generate private key
	priv, _, err := crypto.GenerateKeyPair(keyType, bits)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// Write private key to file in Base 64 format
// Store the key type ID followed by a space, then the key, then a new-line
func StorePrivKeyToFile(priv crypto.PrivKey, keyFile string) error {
	keyFile, err := ExpandTilde(keyFile)
	if err != nil {
		return err
	}

	if FileExists(keyFile) {
		return fmt.Errorf("File already exists (%s).\n"+
			"Delete it or move it before proceeding.", keyFile)
	}

	file, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	rawBytes, err := priv.Raw()
	if err != nil {
		return err
	}

	fileStr := fmt.Sprintf("%d %s\n", priv.Type(), crypto.ConfigEncodeKey(rawBytes))
	_, err = file.WriteString(fileStr)
	if err != nil {
		return err
	}

	return nil
}

// Inverse of storePrivKeyToFile()
func LoadPrivKeyFromFile(keyFile string) (crypto.PrivKey, error) {
	keyFile, err := ExpandTilde(keyFile)
	if err != nil {
		return nil, err
	}

	if !FileExists(keyFile) {
		return nil, fmt.Errorf("File (%s) does not exist.", keyFile)
	}

	/* NOTE: Using ioutil's ReadFile() may be potentially bad in the case that
	 *       the file is very large, as it tries to read the entire file at once.
	 *       Alternative is to read chunk by chunk using os' Read() and combine.
	 *       I'm being lazy, assume file is small or memory is large.
	 */
	content, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	// Strip new-line, then parse key type from key itself
	contentStr := strings.TrimSpace(string(content))
	spaceIdx := strings.IndexByte(contentStr, ' ')
	if spaceIdx <= 0 {
		return nil, fmt.Errorf("Unable to load key file (may have been corrupted)")
	}

	keyType, err := strconv.ParseInt(contentStr[:spaceIdx], 10, 32)
	if err != nil {
		return nil, err
	}

	keyB64 := contentStr[spaceIdx+1:]
	keyRaw, err := crypto.ConfigDecodeKey(keyB64)
	if err != nil {
		return nil, err
	}

	// Unmarsall to create private key object
	unmarshaller, ok := crypto.PrivKeyUnmarshallers[pb.KeyType(keyType)]
	if !ok {
		return nil, fmt.Errorf("Key file contains an unknown algorithm.")
	}

	return unmarshaller(keyRaw)
}

type KeyFlags struct {
	Algo      *string
	Bits      *int
	Keyfile   *string
	Ephemeral *bool
}

// Adds CLI arguments for key-related flags. Does not call Parse() on its own.
//
// Takes a single parameter, the default key filename. This allows programs
// to define different default key filenames, so that if two different
// programs are run on the same host, they won't read from the same key.
//
// Returns a struct containing pointers to the various variables that will
// hold the parsed flag values once Parse() is called.
func AddKeyFlags(defaultKeyFile string) (KeyFlags, error) {
	// Check if Parse() has been called. If it has, adding new flags will not
	// have any effect. Ensure the check only runs when not running go test.
	if flag.Parsed() && !strings.HasSuffix(os.Args[0], ".test") {
		return KeyFlags{}, fmt.Errorf("Already parsed CLI flags, cannot add new flags")
	}

	keyFlags := KeyFlags{}

	keyFlags.Algo = flag.String("algo", "RSA",
		"Cryptographic algorithm to use for generating the key.\n"+
			"Will be ignored if 'genkey' is false.\n"+
			"Must be one of {RSA, Ed25519, Secp256k1, ECDSA}")
	keyFlags.Bits = flag.Int("bits", 2048,
		"Key length, in bits. Will be ignored if 'algo' is not RSA.")
	keyFlags.Keyfile = flag.String("keyfile", defaultKeyFile,
		"Location of private key to read from (or write to, if generating).")
	keyFlags.Ephemeral = flag.Bool("ephemeral", false,
		"Generate a new key just for this run, and don't store it to file.\n"+
			"If 'keyfile' is specified, it will be ignored.")

	return keyFlags, nil
}

// Sanity checking for KeyFlags struct. Ensures it's properly populated
// with valid pointers and non-zero values.
func checkKeyFlags(kf *KeyFlags) error {
	if kf.Algo == nil || kf.Bits == nil ||
		kf.Keyfile == nil || kf.Ephemeral == nil {

		return fmt.Errorf("ERROR: KeyFlags contains nil pointers")
	}

	if *kf.Algo == "" || *kf.Bits == 0 || *kf.Keyfile == "" {
		return fmt.Errorf("ERROR: KeyFlags contains zero-valued variables (check algo, bits, or keyfile)")
	}

	return nil
}

func CreateOrLoadKey(kf KeyFlags) (crypto.PrivKey, error) {
	if err := checkKeyFlags(&kf); err != nil {
		return nil, err
	}

	var priv crypto.PrivKey
	var err error

	// If ephemeral, simplest case
	if *kf.Ephemeral {
		log.Println("Generating a new key...")
		if priv, err = GeneratePrivKey(*kf.Algo, *kf.Bits); err != nil {
			return nil, fmt.Errorf("ERROR: Unable to generate key\n%w", err)
		}

		return priv, nil
	}

	// Not ephemeral, just load the key and return it
	if FileExists(*kf.Keyfile) {
		if priv, err = LoadPrivKeyFromFile(*kf.Keyfile); err != nil {
			return nil, fmt.Errorf("ERROR: Unable to load key from file\n%w", err)
		}

		return priv, nil
	}

	// If key doesn't exist, will have to generate a new one and store it
	log.Printf("Key does not exist at location: %s.\n", *kf.Keyfile)
	log.Println("Generating a new key...")
	if priv, err = GeneratePrivKey(*kf.Algo, *kf.Bits); err != nil {
		return nil, fmt.Errorf("ERROR: Unable to generate key\n%w", err)
	}

	if err = StorePrivKeyToFile(priv, *kf.Keyfile); err != nil {
		return nil, fmt.Errorf("ERROR: Unable to save key to file %s\n%w", *kf.Keyfile, err)
	}
	log.Println("New key is stored at:", *kf.Keyfile)

	return priv, nil
}
