// ABOUTME: Unit tests for vector storage functionality
// ABOUTME: Tests cosine similarity calculation
package storage

import (
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.0, 1.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{-1.0, 0.0, 0.0},
			expected: -1.0,
			delta:    0.001,
		},
		{
			name:     "similar vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.9, 0.1, 0.0},
			expected: 0.995,
			delta:    0.01,
		},
		{
			name:     "different length vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "zero vector",
			a:        []float64{0.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if abs(result-tt.expected) > tt.delta {
				t.Errorf("cosineSimilarity(%v, %v) = %.4f, expected %.4f (delta %.4f)",
					tt.a, tt.b, result, tt.expected, tt.delta)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
