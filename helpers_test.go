package plex

import (
	"fmt"
	"math/rand"
)

// setOfRandBytes returns a set of random byte slices of the given size
func setOfRandBytes(size int) (data [][]byte, err error) {
	data = make([][]byte, size)
	for i := 0; i < len(data); i++ {
		// Mod 1024 to make sure the random bytes
		// are not too large
		data[i], err = randBytes(rand.Int() % 1024)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

//	randBytes returns a random byte slice of the given size
func randBytes(size int) ([]byte, error) {
	buff := make([]byte, size)
	n, err := rand.Read(buff)
	if err != nil {
		return nil, fmt.Errorf("error reading random data: %s", err)
	}

	return buff[:n], nil
}
