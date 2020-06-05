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
	"strings"
	"testing"

	"github.com/Multi-Tier-Cloud/common/util"
)

/*  Keeping this here for manual testing purposes
 *  This custom TestMain enables users to pass the -bootstrap flags when
 *  running the test. E.g.
 *      go test -bootstrap <multiaddr-1> -bootstrap <multiaddr-2>
 */
//func TestMain(test *testing.M) {
//    _, err := util.AddBootstrapFlags()
//    if err != nil {
//        test.Fatalf("Unable to add bootstrap flags")
//    }
//
//    flag.Parse()
//
//    os.Exit(test.Run())
//}

const (
	testMultiAddr1 = "/ip4/10.11.17.15/tcp/4001/ipfs/QmeZvvPZgrpgSLFyTYwCUEbyK6Ks8Cjm2GGrP2PA78zjAk"
	testMultiAddr2 = "/ip4/10.11.17.32/tcp/4001/ipfs/12D3KooWGegi4bWDPw9f6x2mZ6zxtsjR8w4ax1tEMDKCNqdYBt7X"
	testBadAddr    = "/hello/World"
)

func TestAddBootstrapFlags(test *testing.T) {
	// Test calling AddBootstrapFlags() multiple times
	bootstraps, err := util.AddBootstrapFlags()
	if err != nil {
		test.Fatalf("ERROR: Unable to add bootstrap flags")
	}

	bootstraps2, err := util.AddBootstrapFlags()
	if err != nil {
		test.Fatalf("ERROR: Unable to add bootstrap flags")
	}

	if bootstraps != bootstraps2 {
		test.Fatalf("ERROR: Subsequent calls to AddBootstrapFlags() returned " +
			"different values. They should be the same.")
	}
}

func TestBootstrapSetString(test *testing.T) {
	bootstraps := util.GetBootstrapPointer()

	// Test setting a bad address
	err := bootstraps.Set(testBadAddr)
	if err == nil {
		test.Fatalf("ERROR: Sucecssfully set bad address (%s) for bootstrap. "+
			"Expected it to fail", testBadAddr)
	}

	// Test setting a proper address
	err = bootstraps.Set(testMultiAddr1)
	if err != nil {
		test.Fatalf("ERROR: Setting address (%s) failed.", testMultiAddr1)
	}

	if len(*bootstraps) != 1 {
		test.Fatalf("ERROR: Set address (%s), but it did not get added to the list of addresses.", testMultiAddr1)
	}

	// Test setting the same address again.
	err = bootstraps.Set(testMultiAddr1)
	if err != nil {
		test.Fatalf("ERROR: Setting the same address (%s) failed.\n"+
			"Expected setting duplicate addresses to succeed (idempotent).", testMultiAddr1)
	}

	if len(*bootstraps) != 1 {
		test.Fatalf("ERROR: Set address (%s) a second time and the list of "+
			"addresses appears to have changed.", testMultiAddr1)
	}

	err = bootstraps.Set(testMultiAddr2)
	if err != nil {
		test.Fatalf("ERROR: Setting address (%s) failed.", testMultiAddr2)
	}

	if len(*bootstraps) != 2 {
		test.Fatalf("ERROR: Added new address (%s), the list of addresses "+
			"should have increased.", testMultiAddr2)
	}

	// Test printing works
	printTest := bootstraps.String()
	if len(printTest) <= 2 {
		test.Fatalf("ERROR: Expected String() to print the list of bootstrap nodes set."+
			"Received a string length of (%d), was expected > 2.", len(printTest))
	}
}

func TestGetEnvBootstraps(test *testing.T) {
	// Set the environment variable, then call GetEnvBootstraps()
	fakeEnvVal := "\t  /ip4/10.11.69.5/tcp/36277/p2p/QmPqv37ukZLuVKfz5vBaH5KyMR9FCo8FuaRpXg7aKwcsgN\t\n\r   " +
		"/ip4/10.11.69.20/tcp/40863/p2p/Qmaq76Lt4oEiYEbkxwCb6CgKbbp9qw5eWTexsrm84D2hJW\t\t\r\n "
	fakeEnvLength := len(strings.Fields(fakeEnvVal))

	err := os.Setenv(util.ENV_KEY_BOOTSTRAPS, fakeEnvVal)
	if err != nil {
		test.Fatalf("ERROR: Unable to set environment variable %s\n", util.ENV_KEY_BOOTSTRAPS)
	}

	bootstraps, err := util.GetEnvBootstraps()
	if err != nil {
		test.Fatalf("ERROR: Case with environment variable set, "+
			"GetEnvBootstraps() failed with error:\n%v\n", err)
	}

	if len(bootstraps) != fakeEnvLength {
		test.Errorf("ERROR: GetEnvBootstraps() returned %d addresses. Expected "+
			"it to return %d instead\n", len(bootstraps), fakeEnvLength)
	}

	// Unset environment variable and re-test
	err = os.Unsetenv(util.ENV_KEY_BOOTSTRAPS)
	if err != nil {
		test.Fatalf("ERROR: Unable to unset environment variable %s\n", util.ENV_KEY_BOOTSTRAPS)
	}

	bootstraps, err = util.GetEnvBootstraps()
	if err != nil {
		test.Fatalf("ERROR: Case with no environment variable set, "+
			"GetEnvBootstraps() failed with error:\n%v\n", err)
	}

	if len(bootstraps) != 0 {
		test.Errorf("ERROR: GetEnvBootstraps() returned %d addresses. Expected "+
			"none since no environment variable was set\n", len(bootstraps))
	}
}
