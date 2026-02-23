package toolfilter

// LevenshteinDistance computes the Levenshtein edit distance between two strings
// using standard dynamic programming. Comparison is case-sensitive.
func LevenshteinDistance(a, b string) int {
	la := len(a)
	lb := len(b)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows instead of full matrix to save memory.
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := prev[j] + 1
			del := curr[j-1] + 1
			sub := prev[j-1] + cost

			min := ins
			if del < min {
				min = del
			}
			if sub < min {
				min = sub
			}
			curr[j] = min
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// SuggestTool finds the closest tool name from available using Levenshtein
// distance. If the smallest distance is <= 3, it returns that tool name.
// Otherwise it returns an empty string.
func SuggestTool(name string, available []string) string {
	bestDist := -1
	bestName := ""

	for _, t := range available {
		d := LevenshteinDistance(name, t)
		if bestDist < 0 || d < bestDist {
			bestDist = d
			bestName = t
		}
	}

	if bestDist >= 0 && bestDist <= 3 {
		return bestName
	}
	return ""
}
