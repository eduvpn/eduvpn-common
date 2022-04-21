package eduvpn

import (
	"crypto/rand"
	"time"
)

// Creates a random byteslice of `size`
func MakeRandomByteSlice(size int) ([]byte, error) {
	byteSlice := make([]byte, size)
	_, err := rand.Read(byteSlice)
	if err != nil {
		return nil, err
	}
	return byteSlice, nil
}

func GenerateTimeSeconds() int64 {
	current := time.Now()
	return current.Unix()
}
