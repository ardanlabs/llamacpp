package catalog

import (
	"fmt"
	"testing"

	"github.com/ardanlabs/kronk/defaults"
)

func Test_Hack(t *testing.T) {
	basePath := defaults.BaseDir("")

	// err := download(basePath)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	catalogs, err := Retrieve(basePath)
	if err != nil {
		t.Fatal(err)
	}

	for _, catalog := range catalogs {
		fmt.Println("Catalog:", catalog.Name)
		for _, model := range catalog.Models {
			fmt.Println("ID:", model.ID)
		}
	}
}
