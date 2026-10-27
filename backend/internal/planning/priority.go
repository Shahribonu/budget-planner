package planning

import "strings"

var (
	mustCategories      = map[string]bool{"health": true, "rent": true, "utilities": true, "education": true}
	niceCategories      = map[string]bool{"entertainment": true, "travel": true, "beauty": true, "cafe": true}
	essentialCategories = map[string]bool{"health": true, "rent": true, "utilities": true, "education": true, "family": true, "groceries": true}
	// Gadgets and treats people tend to want rather than need.
	wantWords = []string{"airpods", "iphone", "playstation", "наушник", "quloqchin", "smart watch", "apple watch", "macbook", "konsol"}
)

// SuggestPriority guesses a priority from the category and title. The user confirms it.
func SuggestPriority(categoryKey, title string) Priority {
	low := strings.ToLower(title)
	for _, w := range wantWords {
		if strings.Contains(low, w) {
			return Nice
		}
	}
	switch {
	case mustCategories[categoryKey]:
		return Must
	case niceCategories[categoryKey]:
		return Nice
	}
	return Should
}

// Essential reports whether a category is a need (health, housing, food, family, education).
func Essential(categoryKey string) bool { return essentialCategories[categoryKey] }
