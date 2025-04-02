package vec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestEncodeFloat64s(t *testing.T) {
	tests := []struct {
		name   string
		input  []float64
		length int // Expected length of the output in bytes
	}{
		{
			name:   "Empty slice",
			input:  []float64{},
			length: 0,
		},
		{
			name:   "Single value",
			input:  []float64{1.23},
			length: 8,
		},
		{
			name:   "Multiple values",
			input:  []float64{1.23, 4.56, 7.89},
			length: 24,
		},
		{
			name:   "Special values",
			input:  []float64{0.0, -0.0, math.Inf(1), math.Inf(-1), math.NaN()},
			length: 40,
		},
		{
			name:   "Very large and very small values",
			input:  []float64{math.MaxFloat64, math.SmallestNonzeroFloat64},
			length: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeVector(tt.input)

			// Check if length is correct
			if len(result) != tt.length {
				t.Errorf("Expected length %d, got %d", tt.length, len(result))
			}

			// For non-empty slices, verify the output is not all zeros
			if len(result) > 0 && bytes.Equal(result, make([]byte, len(result))) {
				t.Errorf("Expected non-zero encoded bytes, got all zeros")
			}
		})
	}
}

func TestDecodeFloat64s(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		want   []float64
		errMsg string
	}{
		{
			name:   "Empty slice",
			input:  []byte{},
			want:   []float64{},
			errMsg: "",
		},
		{
			name:   "Invalid length",
			input:  []byte{1, 2, 3}, // Not divisible by 8
			want:   nil,
			errMsg: "invalid data length: 3 is not divisible by 8",
		},
		{
			name:   "Single zero value",
			input:  []byte{0, 0, 0, 0, 0, 0, 0, 0},
			want:   []float64{0.0},
			errMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeVector(tt.input)

			// Check error message if expected
			if tt.errMsg != "" {
				if err == nil {
					t.Errorf("Expected error message: %s, got nil", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error message: %s, got: %s", tt.errMsg, err.Error())
				}
				return
			}

			// If no error expected, verify there was none
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			// Check if results match expected output
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected %v, got %v", tt.want, got)
			}
		})
	}
}

// TestRoundTrip verifies that encoding and then decoding returns the original values
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
	}{
		{
			name:  "Empty slice",
			input: []float64{},
		},
		{
			name:  "Single value",
			input: []float64{42.0},
		},
		{
			name:  "Multiple regular values",
			input: []float64{1.23, 4.56, 7.89, -123.456},
		},
		{
			name:  "Special values",
			input: []float64{0.0, -0.0, math.Inf(1), math.Inf(-1)},
		},
		{
			name:  "Edge values",
			input: []float64{math.MaxFloat64, math.SmallestNonzeroFloat64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode the input
			encoded := EncodeVector(tt.input)

			// Decode the encoded bytes
			decoded, err := DecodeVector(encoded)
			if err != nil {
				t.Errorf("Unexpected error during decoding: %v", err)
				return
			}

			// Check if length matches
			if len(decoded) != len(tt.input) {
				t.Errorf("Expected length %d, got %d", len(tt.input), len(decoded))
				return
			}

			// Check if each value matches
			// Note: we need special handling for NaN because NaN != NaN
			for i := 0; i < len(tt.input); i++ {
				if math.IsNaN(tt.input[i]) {
					if !math.IsNaN(decoded[i]) {
						t.Errorf("At index %d: Expected NaN, got %v", i, decoded[i])
					}
				} else if tt.input[i] != decoded[i] {
					t.Errorf("At index %d: Expected %v, got %v", i, tt.input[i], decoded[i])
				}
			}
		})
	}
}

// TestCompareWithStandardEncoding compares our manual implementation with the standard binary package
func TestCompareWithStandardEncoding(t *testing.T) {
	testValues := []float64{1.23, 4.56, 7.89, math.Pi, math.E, 0.0, -0.0, math.MaxFloat64, math.SmallestNonzeroFloat64}

	// Encode using our manual implementation
	manualEncoded := EncodeVector(testValues)

	// Encode using the standard library
	standardEncoded, err := StdEncodeFloat64s(testValues)
	if err != nil {
		t.Errorf("Unexpected error during standard encoding: %v", err)
		return
	}

	// Compare the encoded byte slices
	if !bytes.Equal(manualEncoded, standardEncoded) {
		t.Errorf("Manual encoding doesn't match standard encoding")
		t.Errorf("Manual:   %v", manualEncoded)
		t.Errorf("Standard: %v", standardEncoded)
	}

	// Now decode using our manual implementation
	manualDecoded, err := DecodeVector(standardEncoded)
	if err != nil {
		t.Errorf("Unexpected error during manual decoding: %v", err)
		return
	}

	// Verify decoded values match the originals
	for i, val := range testValues {
		if math.IsNaN(val) {
			if !math.IsNaN(manualDecoded[i]) {
				t.Errorf("At index %d: Expected NaN, got %v", i, manualDecoded[i])
			}
		} else if val != manualDecoded[i] {
			t.Errorf("At index %d: Expected %v, got %v", i, val, manualDecoded[i])
		}
	}
}

// EncodeVector converts a slice of float64 values to a byte slice
func StdEncodeFloat64s(floats []float64) ([]byte, error) {
	// Create a buffer to store the encoded bytes
	buf := new(bytes.Buffer)

	// Write each float64 value as 8 bytes to the buffer
	for _, f := range floats {
		err := binary.Write(buf, binary.LittleEndian, f)
		if err != nil {
			return nil, fmt.Errorf("encoding error: %w", err)
		}
	}

	// Return the buffer as a byte slice
	return buf.Bytes(), nil
}

// DecodeVector converts a byte slice back to a slice of float64 values
func StdDecodeFloat64s(data []byte) ([]float64, error) {
	// Check if the data length is divisible by 8 (each float64 is 8 bytes)
	if len(data)%8 != 0 {
		return nil, fmt.Errorf("invalid data length: %d is not divisible by 8", len(data))
	}

	// Calculate the number of float64 values
	count := len(data) / 8

	// Create a reader from the byte slice
	reader := bytes.NewReader(data)

	// Create a slice to hold the decoded float64 values
	result := make([]float64, count)

	// Read each 8-byte chunk and convert it to a float64
	for i := 0; i < count; i++ {
		err := binary.Read(reader, binary.LittleEndian, &result[i])
		if err != nil {
			return nil, fmt.Errorf("decoding error at index %d: %w", i, err)
		}
	}

	return result, nil
}

func generateTestData() []float64 {
	const size = 1024

	data := make([]float64, size)

	// Fill most of the slice with random values
	for i := 0; i < size-8; i++ {
		data[i] = rand.Float64() * math.Pow(10, rand.Float64()*10-5) // Range from 1e-5 to 1e5
	}

	// Add some special values at the end
	specialValues := []float64{
		0.0,
		-0.0,
		1.0,
		-1.0,
		math.Pi,
		math.E,
		math.Inf(1),
		math.Inf(-1),
		math.NaN(),
		math.MaxFloat64,
		math.SmallestNonzeroFloat64,
		math.Float64frombits(0xFFFFFFFFFFFFFFFF), // All bits set
	}

	copy(data[size-len(specialValues):], specialValues)

	return data
}

func BenchmarkEncodeFloat64s(b *testing.B) {
	testData := generateTestData()

	b.Run("Manual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = EncodeVector(testData)
		}
	})

	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = StdEncodeFloat64s(testData)
		}
	})
}

func BenchmarkDecodeFloat64s(b *testing.B) {
	testData := generateTestData()
	encodedData := EncodeVector(testData)
	stdEncodedData, _ := StdEncodeFloat64s(testData)

	b.Run("Manual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = DecodeVector(encodedData)
		}
	})

	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = StdDecodeFloat64s(stdEncodedData)
		}
	})
}
