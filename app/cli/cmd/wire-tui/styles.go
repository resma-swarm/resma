package main

import "github.com/charmbracelet/lipgloss"

// Cores do tema dark (roxo/cyan)
var (
	cPrimary   = lipgloss.Color("#7D56F3")
	cAccent    = lipgloss.Color("#00D9FF")
	cSuccess   = lipgloss.Color("#04E762")
	cWarning   = lipgloss.Color("#FFB400")
	cError     = lipgloss.Color("#FF5C5C")
	cMuted     = lipgloss.Color("#6B7280")
	cBorder    = lipgloss.Color("#3D3D5C")
	cBorderAct = lipgloss.Color("#7D56F3")
	cBg        = lipgloss.Color("#1A1A2E")
	cTabBg     = lipgloss.Color("#2A2A3E")
	cWhite     = lipgloss.Color("#FFFFFF")
)

// Estilos
var (
	sHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(cWhite).
		Background(cPrimary).
		Padding(0, 2)

	sSubtitle = lipgloss.NewStyle().
			Foreground(cMuted)

	sTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(cWhite).
			Background(cPrimary).
			Padding(0, 2).
			MarginRight(1)

	sTabInactive = lipgloss.NewStyle().
			Foreground(cMuted).
			Background(cTabBg).
			Padding(0, 2).
			MarginRight(1)

	sFooter = lipgloss.NewStyle().
		Foreground(cMuted).
		Padding(0, 1)

	sBreadcrumb = lipgloss.NewStyle().
			Foreground(cMuted).
			Padding(0, 1)

	sTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(cAccent)

	sTableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(cAccent)

	sSuccess = lipgloss.NewStyle().Foreground(cSuccess).Bold(true)
	sWarning = lipgloss.NewStyle().Foreground(cWarning).Bold(true)
	sError   = lipgloss.NewStyle().Foreground(cError).Bold(true)
	sMuted   = lipgloss.NewStyle().Foreground(cMuted)
	sHighlight = lipgloss.NewStyle().Foreground(cPrimary).Bold(true)

	sCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cBorder).
		Padding(0, 2).
		MarginBottom(0)

	sCardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cAccent)

	sBorderActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorderAct)

	sBorderInactive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder)

	sSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(cWhite).
			Background(cPrimary)

	sInput = lipgloss.NewStyle().
		Foreground(cWhite).
		Background(cTabBg).
		Padding(0, 1)

	sSparkline = lipgloss.NewStyle().Foreground(cAccent)
)
