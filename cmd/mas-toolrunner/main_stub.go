//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "mas-toolrunner is only supported on linux")
	os.Exit(1)
}
