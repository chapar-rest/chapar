package fuzzysearch

import (
	"sort"
	"strings"
	"unicode"

	"gioui.org/widget"
)

// Item represents an object with a title that can be searched
type Item struct {
	Identifier string
	Title      string
	Kind       string
}

type SearchResult struct {
	Item  Item
	Score float64

	Icon      *widget.Icon
	Clickable widget.Clickable
}

// FuzzySearch performs fuzzy search on a list of items based on their titles
func FuzzySearch(items []Item, query string, maxResults int) []*SearchResult {
	if len(query) == 0 {
		return nil
	}

	// Normalize the query
	query = normalizeString(query)

	// Calculate scores for each item
	var results []*SearchResult
	for _, item := range items {
		normalizedTitle := normalizeString(item.Title)

		// Skip empty titles
		if len(normalizedTitle) == 0 {
			continue
		}

		// Calculate different matching metrics
		levenScore := levenshteinScore(normalizedTitle, query)
		prefixScore := prefixScore(normalizedTitle, query)
		containsScore := containsScore(normalizedTitle, query)

		// Combine scores (TODO - make these weights configurable)
		finalScore := levenScore*0.6 + prefixScore*0.3 + containsScore*0.1

		// Only include items with a score above a certain threshold
		if finalScore > 0.1 {
			results = append(results, &SearchResult{
				Item:  item,
				Score: finalScore,
			})
		}
	}

	// Sort results by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results if needed
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// normalizeString normalizes a string by converting to lowercase and removing special characters
func normalizeString(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	s1Len := len(s1)
	s2Len := len(s2)

	// Create a matrix of size (s1Len+1) x (s2Len+1)
	matrix := make([][]int, s1Len+1)
	for i := range matrix {
		matrix[i] = make([]int, s2Len+1)
	}

	// Initialize the first row and column
	for i := 0; i <= s1Len; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= s2Len; j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= s1Len; i++ {
		for j := 1; j <= s2Len; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[s1Len][s2Len]
}

// levenshteinScore converts Levenshtein distance to a score between 0 and 1
// where 1 means perfect match and 0 means no match
func levenshteinScore(s1, s2 string) float64 {
	// For empty strings
	if len(s1) == 0 || len(s2) == 0 {
		return 0
	}

	// Calculate max possible distance (length of longer string)
	maxDistance := max(len(s1), len(s2))
	distance := levenshteinDistance(s1, s2)

	// Convert to a score where higher is better
	return 1 - float64(distance)/float64(maxDistance)
}

// prefixScore gives a higher score if the query is a prefix of the title
func prefixScore(title, query string) float64 {
	if strings.HasPrefix(title, query) {
		// Return a score based on how much of the title is matched by the prefix
		return float64(len(query)) / float64(len(title))
	}
	return 0
}

// containsScore gives a score if the title contains the query
func containsScore(title, query string) float64 {
	if strings.Contains(title, query) {
		return 0.8
	}
	return 0
}
