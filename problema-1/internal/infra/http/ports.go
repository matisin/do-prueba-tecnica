package http_adapter

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func isPortAvailable(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, 250*time.Millisecond)
	closed := err != nil
	if !closed {
		conn.Close()
	}
	return closed
}

func randomPort() int {
	maxPort, minPort := 65535, 1024
	return minPort + rand.Intn(maxPort-minPort+1)
}

func getAvailablePort(strPort string) int {
	port, err := strconv.Atoi(strPort)
	maxPort, minPort := 65535, 1024
	if err != nil || port > maxPort || port < minPort {
		port = randomPort()
	}
	if isPortAvailable(port) {
		return port
	}
	for i := 0; i < 5; i++ {
		port = randomPort()
		if isPortAvailable(port) {
			return port
		}
	}
	return 0
}
