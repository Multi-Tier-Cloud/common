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

package util_test

import (
	//"flag"
	"os"
	"reflect"
	"testing"

	"github.com/Multi-Tier-Cloud/common/util"
)

var (
	testPassphrase = "toInfinityAndBeyond!"
)

/*  Keeping this here for manual testing purposes
 *  This custom TestMain enables users to pass the -psk flag when
 *  running the test. E.g.
 *      go test -psk <pre-shared-key-passphrase>
 */
//func TestMain(test *testing.M) {
//    _, err := util.AddPSKFlag()
//    if err != nil {
//        test.Fatalf("Unable to add PSK flag")
//    }
//
//    flag.Parse()
//
//    os.Exit(test.Run())
//}

func TestAddPSKFlag(test *testing.T) {
	psk, err := util.AddPSKFlag()
	if err != nil {
		test.Fatalf("ERROR: Unable to add PSK flag")
	}

	psk2, err := util.AddPSKFlag()
	if err != nil {
		test.Fatalf("ERROR: Unable to add PSK flag")
	}

	if psk != psk2 {
		test.Fatalf("ERROR: Subsequent calls to AddPSKFlag() returned " +
			"different values. They should be the same.")
	}
}

func TestPSKSetString(test *testing.T) {
	psk := util.GetPSKPointer()

	// Test setting with an empty string (should generate a random PSK)
	err := psk.Set("")
	if err != nil {
		test.Fatalf("ERROR: Setting PSK flag with \"\" failed.\n")
	}

	tmp := *psk

	err = psk.Set("") // Generates random PSK again
	if err != nil {
		test.Fatalf("ERROR: Setting PSK flag with \"\" a second time failed.\n")
	}

	if reflect.DeepEqual(tmp, *psk) {
		test.Errorf("ERROR: Expected new random PSK to be generated when Set()" +
			"is called with an empty string \"\", but they were identical.\n")
	}

	// Test setting with same passphrase twice
	err = psk.Set(testPassphrase)
	if err != nil {
		test.Fatalf("ERROR: Setting PSK flag with \"%s\" failed.\n", testPassphrase)
	}

	tmp = *psk

	err = psk.Set(testPassphrase)
	if err != nil {
		test.Fatalf("ERROR: Setting PSK flag with \"%s\" a second time failed.\n", testPassphrase)
	}

	if !reflect.DeepEqual(tmp, *psk) {
		test.Errorf("ERROR: Expected the same PSK to be generated when Set()" +
			"is called twice with the same psk passphrase, but they differed.\n")
	}

	// Test to ensure original passphrase is retrievable
	if util.GetFlagPSKString() != testPassphrase {
		test.Errorf("ERROR: GetFlagPSKString() returned a passphrase that " +
			"differs from the original passphrase\n")
	}

	// Test to ensure hashed PSK is the correct length
	hPsk, err := util.AddPSKFlag()
	if err != nil {
		test.Fatalf("ERROR: Unable to add PSK flag")
	}
	if len(*hPsk) != util.PSK_NUM_BYTES {
		test.Fatalf("ERROR: Expected PSK String() to return a 32-character value, "+
			"but it returned a %d characters instead.\n", len(*hPsk))
	}

	// Test printing works and ensure it matches original passphrase
	printTest := psk.String()
	if printTest != testPassphrase {
		test.Errorf("ERROR: Expected GetPSKString() to return the original passphrase\n")
	}
}

func TestGetEnvPSK(test *testing.T) {
	// Set the environment variable, then call GetEnvPSK()
	fakeEnvVal := "\t  helloWorldPassphrase     \r\n\t "

	err := os.Setenv(util.ENV_KEY_PSK, fakeEnvVal)
	if err != nil {
		test.Fatalf("ERROR: Unable to set environment variable %s\n", util.ENV_KEY_PSK)
	}

	psk, err := util.GetEnvPSK()
	if err != nil {
		test.Fatalf("ERROR: Case with environment variable set, "+
			"GetEnvPSK() failed with error:\n%v\n", err)
	}

	if len(psk) != util.PSK_NUM_BYTES {
		test.Errorf("ERROR: GetEnvPSK() returned a key with length %d. "+
			"Expected the length to be %d\n", len(psk), util.PSK_NUM_BYTES)
	}

	// Test to ensure original passphrase is retrievable
	if util.GetEnvPSKString() != fakeEnvVal {
		test.Errorf("ERROR: GetEnvPSKString() returned a passphrase that " +
			"differs from the original passphrase\n")
	}

	// Unset environment variable and re-test
	err = os.Unsetenv(util.ENV_KEY_PSK)
	if err != nil {
		test.Fatalf("ERROR: Unable to unset environment variable %s\n", util.ENV_KEY_PSK)
	}

	psk, err = util.GetEnvPSK()
	if err != nil {
		test.Fatalf("ERROR: Case with no environment variable set, "+
			"GetEnvPSK() failed with error:\n%v\n", err)
	}

	if len(psk) != 0 {
		test.Errorf("ERROR: GetEnvPSK() returned a key with length %d. Expected "+
			"length 0 since no environment variable was set\n", len(psk))
	}
}
