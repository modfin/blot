package vec

import (
	"database/sql/driver"
	"fmt"
	"log/slog"
	"math"
	"modernc.org/sqlite"
	"sync/atomic"
	"time"
)

var vec_dist_tot = atomic.Int64{}
var vec_dist_mar = atomic.Int64{}
var vec_dist_comp = atomic.Int64{}
var vec_dist_count = atomic.Int64{}

func Statistics() {

	if vec_dist_count.Load() > 0 {
		avg := time.Duration(vec_dist_tot.Load() / vec_dist_count.Load())
		slog.Default().Debug("vec_dist comparison stats",
			"vec_dist_count", vec_dist_count.Load(),
			"vec_dist_tot", time.Duration(vec_dist_tot.Load()),
			"marshaling", time.Duration(vec_dist_mar.Load()),
			"comparison", time.Duration(vec_dist_comp.Load()),
			"avg", avg)
	}

}

func init() {

	sqlite.MustRegisterDeterministicScalarFunction("vec_dist", 2, func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		start := time.Now()
		defer func() {
			vec_dist_tot.Add(int64(time.Since(start)))
			vec_dist_count.Add(1)
		}()

		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}

		leftbin, ok := args[0].([]uint8)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		rightbin, ok := args[1].([]uint8)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		unmarshalStart := time.Now()

		left, err := DecodeFloat64s(leftbin)
		if err != nil {
			return nil, err
		}

		right, err := DecodeFloat64s(rightbin)
		if err != nil {
			return nil, err
		}

		unmarshalTime := time.Since(unmarshalStart)
		vec_dist_mar.Add(int64(unmarshalTime))

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
		vec_dist_comp.Add(int64(comparisonTime))

		// Prevent division by zero
		if normA == 0 || normB == 0 {
			return 0.0, nil
		}

		result := -(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))

		return result, nil
	})

}
