package util

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

// Creates a random byteslice of `size`
func MakeRandomByteSlice(size int) ([]byte, error) {
	byteSlice := make([]byte, size)
	_, err := rand.Read(byteSlice)
	if err != nil {
		return nil, &types.WrappedErrorMessage{Message: "failed reading random", Err: err}
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
		return &types.WrappedErrorMessage{Message: fmt.Sprintf("failed to create directory %s", directory), Err: mkdirErr}
	}
	return nil
}
