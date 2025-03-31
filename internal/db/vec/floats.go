package vec

import (
	"errors"
	"strconv"
)

func unmarshalFloats(jsonStr string) ([]float64, error) {
	// Avoid trimming the whole string, just check bounds manually
	startIdx, endIdx := 0, len(jsonStr)-1

	// Skip leading whitespace
	for startIdx < len(jsonStr) && isWhitespace(jsonStr[startIdx]) {
		startIdx++
	}

	// Skip trailing whitespace
	for endIdx > startIdx && isWhitespace(jsonStr[endIdx]) {
		endIdx--
	}

	// Check for array brackets
	if startIdx >= endIdx || jsonStr[startIdx] != '[' || jsonStr[endIdx] != ']' {
		return nil, errors.New("input is not a JSON array")
	}

	// Remove the surrounding brackets
	startIdx++
	endIdx--

	// Preallocate result slice - estimate number of values by counting commas
	commaCount := 0
	for i := startIdx; i <= endIdx; i++ {
		if jsonStr[i] == ',' {
			commaCount++
		}
	}
	result := make([]float64, 0, commaCount+1)

	// Avoid using strings.Builder, use byte slices directly
	var numStart int
	inNumber := false

	for i := startIdx; i <= endIdx; i++ {
		char := jsonStr[i]

		switch {
		// Digits, decimal point, or signs
		case char >= '0' && char <= '9' || char == '.' || char == '-' || char == 'e' || char == 'E' || char == '+':
			if !inNumber {
				numStart = i
				inNumber = true
			}

		// Whitespace or comma
		case isWhitespace(char) || char == ',':
			if inNumber {
				// Parse the number directly from the substring
				num, err := strconv.ParseFloat(jsonStr[numStart:i], 64)
				if err != nil {
					return nil, err
				}
				result = append(result, num)
				inNumber = false
			}

		default:
			return nil, errors.New("invalid character in JSON array")
		}
	}

	// Handle last number if we ended on a number
	if inNumber {
		num, err := strconv.ParseFloat(jsonStr[numStart:endIdx+1], 64)
		if err != nil {
			return nil, err
		}
		result = append(result, num)
	}

	return result, nil
}

// Inline helper function for whitespace checking
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
