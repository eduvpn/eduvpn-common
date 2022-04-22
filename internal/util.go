package internal

import (
	"crypto/rand"
	"os"
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

func EnsureDirectory(directory string) error {
	mkdirErr := os.MkdirAll(directory, os.ModePerm)
	if mkdirErr != nil {
		return mkdirErr
	}
	return nil
}
