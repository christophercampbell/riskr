package main

import (
	"fmt"
	"os"
)

func main() {
	err := commands.Run(os.Args)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
}
