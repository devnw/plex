package plex

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
)

// SumByteSlice is a map with a key which is the sha1sum of the byte slice
type SumByteSlice map[string][]byte

func (s SumByteSlice) SliceOfReadWriter() []io.ReadWriter {
	rws := make([]io.ReadWriter, 0, len(s))
	for _, v := range s {
		rws = append(rws, bytes.NewBuffer(v))
	}

	return rws
}

// setOfRandBytes returns a set of random byte slices of the given size
func setOfRandBytes(size int) (data SumByteSlice, err error) {
	data = SumByteSlice{}

	for i := 0; i < size; i++ {
		// Mod 1024 to make sure the random byteData
		// are not too large
		byteData, err := randBytes(rand.Int() % 1024)
		if err != nil {
			return nil, err
		}

		data[fmt.Sprintf("%x", sha1.Sum(byteData))] = byteData
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
