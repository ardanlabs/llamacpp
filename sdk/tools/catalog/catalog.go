// Package catalog provides tooling support for the catalog system.
package catalog

import (
	"net"
	"time"
)

const (
	localFolder = "catalogs"
	indexFile   = ".index.yaml"
)

func hasNetwork() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return false
	}

	conn.Close()

	return true
}
