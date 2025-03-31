package db

import (
	"database/sql/driver"
	"fmt"
	"github.com/goccy/go-json"
	"log/slog"
	"math"
	"modernc.org/sqlite"
	"sync/atomic"
	"time"
)

var tot = atomic.Int64{}
var count = atomic.Int64{}

func Statistics() {
	if count.Load() == 0 {
		return
	}
	avg := time.Duration(tot.Load() / count.Load())
	slog.Default().Debug("vec_dist comparison stats", "count", count.Load(), "tot", time.Duration(tot.Load()), "avg", avg)
}

func init() {

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

		var left []float64
		err := json.Unmarshal([]byte(leftString), &left)
		if err != nil {
			return nil, err
		}

		var right []float64
		err = json.Unmarshal([]byte(rightString), &right)
		if err != nil {
			return nil, err
		}
		if len(left) != len(right) {
			return nil, fmt.Errorf("expected equal length arrays, got %d and %d", len(left), len(right))
		}

		var dotProduct float64
		var normA float64
		var normB float64

		for i := 0; i < min(len(left), len(right)); i++ {
			dotProduct += left[i] * right[i]
			normA += left[i] * left[i]
			normB += right[i] * right[i]
		}

		// Prevent division by zero
		if normA == 0 || normB == 0 {
			return 0.0, nil
		}

		return -(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))), nil

	})
}
