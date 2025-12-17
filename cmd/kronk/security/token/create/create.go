// Package create provides the token create command code.
package create

import "fmt"

type Config struct {
	AdminToken string
	Duration   string
	Endpoints  []string
}

func runWeb(cfg Config) error {
	fmt.Println("RunWeb: token create")
	fmt.Printf("  AdminToken: %s\n", cfg.AdminToken)
	fmt.Printf("  Duration: %s\n", cfg.Duration)
	fmt.Printf("  Endpoints: %v\n", cfg.Endpoints)

	return nil
}

func runLocal(cfg Config) error {
	fmt.Println("RunLocal: token create")
	fmt.Printf("  AdminToken: %s\n", cfg.AdminToken)
	fmt.Printf("  Duration: %s\n", cfg.Duration)
	fmt.Printf("  Endpoints: %v\n", cfg.Endpoints)

	return nil
}
