// Package templates provides template support.
package templates

import (
	"net"
	"time"
)

const (
	localFolder = "templates"
)

func hasNetwork() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return false
	}

	conn.Close()

	return true
}
