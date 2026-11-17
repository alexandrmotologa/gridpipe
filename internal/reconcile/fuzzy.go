package reconcile

import (
	"math"
	"strings"
	"unicode"
)

// NormalizeKey removes punctuation and extra whitespace for fuzzy comparison.
func NormalizeKey(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// LevenshteinDistance computes the minimum edit distance between two strings.
func LevenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = min(
				dp[i-1][j]+1,      // deletion
				dp[i][j-1]+1,      // insertion
				dp[i-1][j-1]+cost, // substitution
			)
		}
	}

	return dp[n][m]
}

// JaroSimilarity computes the Jaro similarity metric (0.0 to 1.0).
func JaroSimilarity(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)
	if len1 == 0 && len2 == 0 {
		return 1.0
	}
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	matchDistance := max(len1, len2)/2 - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)
	matches := 0

	for i := 0; i < len1; i++ {
		start := max(0, i-matchDistance)
		end := min(i+matchDistance+1, len2)

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	transpositions := 0
	k := 0
	for i := 0; i < len1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(len1) + m/float64(len2) + (m-float64(transpositions/2))/m) / 3.0
}

// JaroWinklerSimilarity gives a higher weight to prefix matches.
func JaroWinklerSimilarity(s1, s2 string) float64 {
	jaro := JaroSimilarity(s1, s2)
	if jaro < 0.7 {
		return jaro
	}

	r1, r2 := []rune(s1), []rune(s2)
	prefixLen := 0
	maxPrefix := min(4, min(len(r1), len(r2)))

	for i := 0; i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	scalingFactor := 0.1
	return jaro + float64(prefixLen)*scalingFactor*(1.0-jaro)
}

// SimilarityScore computes overall fuzzy similarity combining normalized and raw Jaro-Winkler.
func SimilarityScore(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	norm1 := NormalizeKey(s1)
	norm2 := NormalizeKey(s2)
	if norm1 == norm2 && norm1 != "" {
		return 0.98 // matched after whitespace/punctuation normalization
	}

	rawScore := JaroWinklerSimilarity(s1, s2)
	normScore := JaroWinklerSimilarity(norm1, norm2)

	return math.Max(rawScore, normScore)
}
