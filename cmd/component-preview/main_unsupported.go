//go:build !darwin && !windows && !(js && wasm)

package main

import "log"

func main() {
	log.Fatal("component previews currently require macOS, Windows, or js/wasm; Linux canvas rendering is not implemented")
}
