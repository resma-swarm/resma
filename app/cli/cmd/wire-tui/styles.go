package main

import "github.com/charmbracelet/lipgloss"

// Cores do k9s default skin
var (
	cK9sBlack   = lipgloss.Color("#000000")
	cK9sGray    = lipgloss.Color("#A9A9A9")
	cK9sMuted   = lipgloss.Color("#808080")
	cK9sAqua    = lipgloss.Color("#00FFFF")
	cK9sCyan    = lipgloss.Color("#00CED1")
	cK9sGreen   = lipgloss.Color("#00FF00")
	cK9sRed     = lipgloss.Color("#FF0000")
	cK9sWarning = lipgloss.Color("#FFA500")
	cK9sWhite   = lipgloss.Color("#FFFFFF")
	cK9sCursor  = lipgloss.Color("#4B0082") // Indigo cursor
	cK9sBorder  = lipgloss.Color("#3D3D5C")
	cK9sPrimary = lipgloss.Color("#7D56F3")
	cK9sTabBg   = lipgloss.Color("#2A2A3E")
)

// Estilos K9s High Fidelity
var (
	sK9sHeader = lipgloss.NewStyle().
			Padding(0, 1)

	sK9sClusterTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cK9sAqua)

	sK9sInfoKey = lipgloss.NewStyle().
			Foreground(cK9sGray)

	sK9sInfoVal = lipgloss.NewStyle().
			Bold(true).
			Foreground(cK9sWhite)

	sK9sMenuKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(cK9sCyan)

	sK9sMenuDesc = lipgloss.NewStyle().
			Foreground(cK9sGray)

	sK9sLogo = lipgloss.NewStyle().
			Foreground(cK9sPrimary).
			Bold(true)

	sK9sStatus = lipgloss.NewStyle().
			Bold(true).
			Foreground(cK9sBlack).
			Padding(0, 2).
			AlignHorizontal(lipgloss.Center)

	sK9sTableCursor = lipgloss.NewStyle().
			Background(cK9sCursor).
			Foreground(cK9sWhite).
			Bold(true)

	sK9sTableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(cK9sAqua).
			Underline(true)

	sK9sCrumbActive = lipgloss.NewStyle().
			Bold(true).
			Background(cK9sPrimary).
			Foreground(cK9sWhite)

	sK9sCrumbInactive = lipgloss.NewStyle().
				Background(cK9sBorder).
				Foreground(cK9sGray)

	sK9sFlash = lipgloss.NewStyle().
			Bold(true).
			AlignHorizontal(lipgloss.Center)

	// Fallbacks para compatibilidade com código existente
	sSuccess   = lipgloss.NewStyle().Foreground(cK9sGreen).Bold(true)
	sWarning   = lipgloss.NewStyle().Foreground(cK9sWarning).Bold(true)
	sError     = lipgloss.NewStyle().Foreground(cK9sRed).Bold(true)
	sMuted     = lipgloss.NewStyle().Foreground(cK9sMuted)
	sHighlight = lipgloss.NewStyle().Foreground(cK9sPrimary).Bold(true)
)
