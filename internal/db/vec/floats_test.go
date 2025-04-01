package vec

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestParseJSONFloatArray(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Empty array",
			input:   "[]",
			wantErr: false,
		},
		{
			name:    "Array with single integer",
			input:   "[42]",
			wantErr: false,
		},
		{
			name:    "Array with single float",
			input:   "[3.14159]",
			wantErr: false,
		},
		{
			name:    "Multiple integers",
			input:   "[1, 2, 3, 4, 5]",
			wantErr: false,
		},
		{
			name:    "Mixed integers and floats",
			input:   "[1, 2.5, -3.7, 4, 5.0]",
			wantErr: false,
		},
		{
			name:    "Scientific notation",
			input:   "[1.2e3, 4.5e-2, 6.7E+1]",
			wantErr: false,
		},
		{
			name:    "Various spacing",
			input:   "[  1.2 ,\n  3.4\t,   5.6   ]",
			wantErr: false,
		},
		{
			name:    "Invalid JSON - no brackets",
			input:   "1, 2, 3",
			wantErr: true,
		},
		{
			name:    "Invalid JSON - non-numeric value",
			input:   "[1, \"hello\", 3]",
			wantErr: true,
		},
		{
			name:    "Invalid JSON - malformed number",
			input:   "[1, 2..5, 3]",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse with our custom function
			customResult, customErr := unmarshalFloats(tc.input)

			// Parse with encoding/json
			var stdResult []float64
			stdErr := json.Unmarshal([]byte(tc.input), &stdResult)

			// Check error conditions
			if tc.wantErr {
				if customErr == nil {
					t.Errorf("parseJSONFloatArray() should have returned an error for input: %s", tc.input)
				}
				return
			}

			// Should not have errors for valid inputs
			if customErr != nil {
				t.Errorf("parseJSONFloatArray() error = %v, input: %s", customErr, tc.input)
				return
			}

			if stdErr != nil {
				t.Errorf("json.Unmarshal() error = %v, input: %s", stdErr, tc.input)
				return
			}

			// Compare results
			if !reflect.DeepEqual(customResult, stdResult) {
				t.Errorf("parseJSONFloatArray() = %v, want %v for input: %s", customResult, stdResult, tc.input)
			}
		})
	}
}

func FuzzParseJSONFloatArray(f *testing.F) {
	// Seed corpus with some well-formed inputs
	seeds := []string{
		"[]",
		"[0]",
		"[1.5]",
		"[-2.718]",
		"[1, 2, 3]",
		"[1.2, 3.4, 5.6]",
		"[1e2, 3.4e-5, 6.7e+8]",
		"[  1.2 ,\n  3.4\t,   5.6   ]",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Skip extremely long inputs to avoid performance issues
		if len(input) > 10000 {
			return
		}

		// Try our custom parser
		customResult, customErr := unmarshalFloats(input)

		// Try standard library parser
		var stdResult []float64
		stdErr := json.Unmarshal([]byte(input), &stdResult)

		// Both should either succeed or fail together for valid JSON
		if (customErr == nil) != (stdErr == nil) {
			// If only one parser returned an error, check if it's a real valid JSON array
			// Some inputs might technically be parseable but not valid JSON
			var testVal interface{}
			if json.Unmarshal([]byte(input), &testVal) == nil {
				// If it is valid JSON, then check if it's actually an array of numbers
				arr, isArray := testVal.([]interface{})
				if isArray {
					allNumbers := true
					for _, v := range arr {
						_, isFloat := v.(float64)
						if !isFloat {
							allNumbers = false
							break
						}
					}

					if allNumbers {
						t.Errorf("Parsing disagreement for valid JSON float array: %q\nCustom error: %v\nStandard error: %v",
							truncateString(input, 100), customErr, stdErr)
					}
				}
			}
			return
		}

		// If both parsers succeeded, results should match
		if customErr == nil && stdErr == nil {
			if !reflect.DeepEqual(customResult, stdResult) {
				t.Errorf("Results don't match for input: %q\nCustom: %v\nStandard: %v",
					truncateString(input, 100), customResult, stdResult)
			}
		}
	})
}

// Helper function to truncate long strings for error messages
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..." // Truncate with ellipsis
}

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
			result := EncodeFloat64s(tt.input)

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
			got, err := DecodeFloat64s(tt.input)

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
			encoded := EncodeFloat64s(tt.input)

			// Decode the encoded bytes
			decoded, err := DecodeFloat64s(encoded)
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
	manualEncoded := EncodeFloat64s(testValues)

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
	manualDecoded, err := DecodeFloat64s(standardEncoded)
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

// EncodeFloat64s converts a slice of float64 values to a byte slice
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

// DecodeFloat64s converts a byte slice back to a slice of float64 values
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
