package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello")
	if len(os.Args) > 0 {
		for i, arg := range os.Args {
			if arg == "--verbose" {
				if i > 0 {
					fmt.Println("verbose mode")
				}
			}
		}
	}
}
