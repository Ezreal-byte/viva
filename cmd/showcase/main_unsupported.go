//go:build !darwin && !windows

package main

import "log"

func main() {
	log.Fatal("showcase requires a supported native desktop backend")
}
