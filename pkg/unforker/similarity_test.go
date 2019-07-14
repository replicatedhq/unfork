package unforker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_keywordComparison(t *testing.T) {
	tests := []struct {
		name   string
		a      []string
		b      []string
		expect int
	}{
		{
			name:   "no overlap",
			a:      []string{"a", "b", "c"},
			b:      []string{"x", "y", "z"},
			expect: 0,
		},
		{
			name:   "1 / 2 overlap",
			a:      []string{"a", "b"},
			b:      []string{"b", "c"},
			expect: 33,
		},
		{
			name:   "1 / 4 overlap",
			a:      []string{"a", "b", "c", "d"},
			b:      []string{"d", "e", "f"},
			expect: 16,
		},
		{
			name:   "it's a match",
			a:      []string{"fuji", "honeycrisp"},
			b:      []string{"apples", "fuji", "honeycrisp"},
			expect: 66,
		},
		{
			name:   "...math",
			a:      []string{"fuji", "honeycrisp", "trader joes", "whole foods", "tempeh", "coffee"},
			b:      []string{"watermelon", "pear", "honeycrisp", "sweetgreens"},
			expect: 11,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := keywordComparison(test.a, test.b)
			req.NoError(err)
			assert.Equal(t, test.expect, actual)
		})
	}
}
