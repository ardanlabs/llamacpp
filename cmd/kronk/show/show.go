// Package show provides the show command code.
package show

import (
	"fmt"
	"time"

	"github.com/ardanlabs/kronk/defaults"
	"github.com/ardanlabs/kronk/tools"
)

// RunLocal executes the pull command.
func RunLocal(args []string) error {
	libPath := defaults.LibsDir("")
	modelPath := defaults.ModelsDir("")
	modelName := args[0]

	mi, err := tools.ShowModel(libPath, modelPath, modelName)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("ID:          %s\n", mi.ID)
	fmt.Printf("Object:      %s\n", mi.Object)
	fmt.Printf("Created:     %v\n", time.UnixMilli(mi.Created))
	fmt.Printf("OwnedBy:     %s\n", mi.OwnedBy)
	fmt.Printf("Desc:        %s\n", mi.Details.Desc)
	fmt.Printf("Size:        %.2f MiB\n", float64(mi.Details.Size)/(1024*1024))
	fmt.Printf("HasProj:     %t\n", mi.Details.HasProjection)
	fmt.Printf("HasEncoder:  %t\n", mi.Details.HasEncoder)
	fmt.Printf("HasDecoder:  %t\n", mi.Details.HasDecoder)
	fmt.Printf("IsRecurrent: %t\n", mi.Details.IsRecurrent)
	fmt.Printf("IsHybrid:    %t\n", mi.Details.IsHybrid)
	fmt.Printf("IsGPT:       %t\n", mi.Details.IsGPT)
	fmt.Println("Metadata:")
	for k, v := range mi.Details.Metadata {
		fmt.Printf("  %s: %s\n", k, v)
	}

	return nil
}
