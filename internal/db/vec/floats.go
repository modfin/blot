package vec

import (
	"fmt"
	"math"
)

// Manual encoding/decoding implementations without using binary package
func EncodeFloat64s(floats []float64) []byte {
	// Allocate a byte slice of the correct size
	result := make([]byte, len(floats)*8)

	for i, f := range floats {
		// Convert float64 to uint64 bits
		bits := math.Float64bits(f)

		// Position in the byte slice for this float
		pos := i * 8

		// Store the uint64 as 8 bytes in little-endian order
		result[pos+0] = byte(bits)
		result[pos+1] = byte(bits >> 8)
		result[pos+2] = byte(bits >> 16)
		result[pos+3] = byte(bits >> 24)
		result[pos+4] = byte(bits >> 32)
		result[pos+5] = byte(bits >> 40)
		result[pos+6] = byte(bits >> 48)
		result[pos+7] = byte(bits >> 56)
	}

	return result
}

func DecodeFloat64s(data []byte) ([]float64, error) {
	// Check if the data length is divisible by 8
	if len(data)%8 != 0 {
		return nil, fmt.Errorf("invalid data length: %d is not divisible by 8", len(data))
	}

	// Calculate the number of float64 values
	count := len(data) / 8

	// Create a slice to hold the decoded values
	result := make([]float64, count)

	for i := 0; i < count; i++ {
		// Position in the byte slice for this float
		pos := i * 8

		// Convert 8 bytes to uint64 in little-endian order
		bits := uint64(data[pos+0]) |
			uint64(data[pos+1])<<8 |
			uint64(data[pos+2])<<16 |
			uint64(data[pos+3])<<24 |
			uint64(data[pos+4])<<32 |
			uint64(data[pos+5])<<40 |
			uint64(data[pos+6])<<48 |
			uint64(data[pos+7])<<56

		// Convert uint64 bits to float64
		result[i] = math.Float64frombits(bits)
	}

	return result, nil
}
