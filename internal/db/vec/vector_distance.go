package vec

import (
	"crypto/sha1"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math"
	"modernc.org/sqlite"
	"sync/atomic"
	"time"
)

var tot = atomic.Int64{}
var mar = atomic.Int64{}
var comp = atomic.Int64{}
var count = atomic.Int64{}

func Statistics() {
	if count.Load() == 0 {
		return
	}
	avg := time.Duration(tot.Load() / count.Load())
	slog.Default().Debug("vec_dist comparison stats",
		"count", count.Load(),
		"tot", time.Duration(tot.Load()),
		"marshaling", time.Duration(mar.Load()),
		"comparison", time.Duration(comp.Load()),
		"avg", avg)
}

func hash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func init() {

	cache := make(map[string][]float64)

	sqlite.MustRegisterDeterministicScalarFunction("vec_dist", 2, func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		start := time.Now()
		defer func() {
			tot.Add(int64(time.Since(start)))
			count.Add(1)
		}()

		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}

		leftString, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		rightString, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		unmarshalStart := time.Now()

		var err error

		left, ok := cache[leftString]
		if !ok {
			left, err = unmarshalFloats(leftString)
			if err != nil {
				return nil, err
			}
			cache[leftString] = left
		}

		right, ok := cache[rightString]
		if !ok {
			right, err = unmarshalFloats(rightString)
			if err != nil {
				return nil, err
			}
			cache[rightString] = right
		}

		unmarshalTime := time.Since(unmarshalStart)
		mar.Add(int64(unmarshalTime))

		if len(left) != len(right) {
			return nil, fmt.Errorf("expected equal length arrays, got %d and %d", len(left), len(right))
		}

		comparisonStart := time.Now()
		var dotProduct float64
		var normA float64
		var normB float64

		for i := 0; i < min(len(left), len(right)); i++ {
			dotProduct += left[i] * right[i]
			normA += left[i] * left[i]
			normB += right[i] * right[i]
		}
		comparisonTime := time.Since(comparisonStart)
		comp.Add(int64(comparisonTime))

		// Prevent division by zero
		if normA == 0 || normB == 0 {
			return 0.0, nil
		}

		result := -(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))

		return result, nil
	})
}
