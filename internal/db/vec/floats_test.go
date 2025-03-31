package vec

import (
	"encoding/json"
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
