package util

import (
	"net"
	"os"
	"strings"
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

