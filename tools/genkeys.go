// To run: go run genkeys.go

package main

import (
	"crypto/rand"
	"fmt"
	"io"
)

// http://www.gorillatoolkit.org/pkg/securecookie#GenerateRandomKey
func GenerateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func main() {
	fmt.Println(fmt.Sprintf("hashKey: %v", GenerateRandomKey(64)))
	fmt.Println(fmt.Sprintf("blockKey: %v", GenerateRandomKey(32)))
}
