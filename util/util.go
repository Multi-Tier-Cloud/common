package util

import (
    "net"
)

// Get preferred outbound ip of this machine
func GetIPAddress() (string, error) {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return "", err
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP.String(), nil
}

func GetFreePort() (int, error) {
	l, err := net.Listen("tcp", "[::]:0")
	if err != nil {
		return 0, err
	}
    defer l.Close()

    port := l.Addr().(*net.TCPAddr).Port

	return port, nil
}
