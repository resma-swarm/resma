package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FlashLevel representa o nível de uma mensagem flash.
type FlashLevel int

const (
	FlashInfo FlashLevel = iota
	FlashWarn
	FlashErr
	FlashSuccess
)

// flashMessage guarda a mensagem e seu nível.
type flashMessage struct {
	text  string
	level FlashLevel
}

// renderFlash renderiza a linha de flash/toast (1 linha, centro-alinhado).
func renderFlash(m model) string {
	if m.flash.text == "" {
		return ""
	}

	emoji := flashEmoji(m.flash.level)
	color := flashColor(m.flash.level)

	text := fmt.Sprintf("%s %s", emoji, m.flash.text)
	return lipgloss.NewStyle().
		Width(m.width).
		Foreground(color).
		AlignHorizontal(lipgloss.Center).
		Render(text)
}

func flashEmoji(l FlashLevel) string {
	switch l {
	case FlashWarn:
		return "⚠"
	case FlashErr:
		return "✗"
	case FlashSuccess:
		return "✓"
	default:
		return "ℹ"
	}
}

func flashColor(l FlashLevel) lipgloss.Color {
	switch l {
	case FlashWarn:
		return cResmaWarning
	case FlashErr:
		return cResmaRed
	case FlashSuccess:
		return cResmaGreen
	default:
		return cResmaCyan
	}
}

// flashText helper para criar mensagem flash.
func flashText(text string, level FlashLevel) flashMessage {
	return flashMessage{text: text, level: level}
}

// renderPrompt renderiza o prompt de command/filter.
func renderPrompt(m model) string {
	if m.viewMode != ViewCommand && m.viewMode != ViewFilter {
		return ""
	}

	var prefix, icon, suggestion string
	var borderColor lipgloss.Color

	if m.viewMode == ViewCommand {
		prefix = ">"
		icon = "🐶"
		borderColor = cResmaPrimary
		suggestion = commandSuggestion(m.inputBuf)
	} else {
		prefix = "/"
		icon = "🐩"
		borderColor = cResmaAqua
	}

	text := m.inputBuf
	if suggestion != "" {
		sugStyled := sMuted.Render(suggestion)
		text = m.inputBuf + sugStyled
	}

	promptStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	content := fmt.Sprintf("%s %s %s_", icon, prefix, text)
	return promptStyle.Render(content)
}

// commandSuggestion retorna a primeira sugestão que começa com o input.
func commandSuggestion(input string) string {
	suggestions := []string{
		"services", "nodes", "agents", "tasks", "alerts", "recs",
		"apply", "rollback", "filter", "quit", "help", "refresh",
	}
	for _, s := range suggestions {
		if strings.HasPrefix(s, input) && s != input {
			return s[len(input):]
		}
	}
	return ""
}
