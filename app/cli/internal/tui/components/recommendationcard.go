package components

import "github.com/charmbracelet/lipgloss"

// RecommendationCard displays a single recommendation with rationale.
type RecommendationCard struct {
	title       string
	description string
	rationale   string
	style       lipgloss.Style
}

// NewRecommendationCard creates a new RecommendationCard component.
func NewRecommendationCard() *RecommendationCard {
	return &RecommendationCard{}
}

// View renders the recommendation card as a string.
func (r RecommendationCard) View() string {
	return ""
}
