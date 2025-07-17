package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Command-line arguments:", os.Args[1:])
	} else {
		fmt.Println("No command-line arguments provided.")
	}
}
