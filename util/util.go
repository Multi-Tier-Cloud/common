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
	"net"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/multiformats/go-multiaddr"
)

// Get preferred outbound ip on machine
func GetIPAddress() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

// Get free port on machine
func GetFreePort() (int, error) {
	l, err := net.Listen("tcp", "[::]:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	return port, nil
}

// Expands tilde to absolute path
// Currently only works if path begins with tilde, not somewhere in the middle
func ExpandTilde(path string) (string, error) {
	newPath := path

	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err != nil {
			return "", err
		} else {
			newPath = home + path[1:]
		}
	}

	return newPath, nil
}

func FileExists(filePath string) bool {
	filePath, err := ExpandTilde(filePath)
	if err != nil {
		return false
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) || info.IsDir() {
		return false
	}

	return true
}

// Helper function to cast a slice of strings into a slice of Multiaddrs
func StringsToMultiaddrs(stringMultiaddrs []string) ([]multiaddr.Multiaddr, error) {
	multiaddrs := make([]multiaddr.Multiaddr, 0)

	for _, s := range stringMultiaddrs {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			return multiaddrs, err
		}
		multiaddrs = append(multiaddrs, ma)
	}

	return multiaddrs, nil
}

// Return identifying multiaddrs of the given node
func Whoami(node host.Host) ([]multiaddr.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		return nil, err
	}

	return addrs, nil
}
