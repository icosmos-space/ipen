package main

import (
	"fmt"
	"os"

	"github.com/icosmos-space/ipen/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
