package tui

import "github.com/charmbracelet/lipgloss"

// Cores da paleta RESMA Dashboard
var (
	cResmaBlack   = lipgloss.Color("#000000")
	cResmaGray    = lipgloss.Color("#A9A9A9")
	cResmaMuted   = lipgloss.Color("#808080")
	cResmaAqua    = lipgloss.Color("#00FFFF")
	cResmaCyan    = lipgloss.Color("#00CED1")
	cResmaGreen   = lipgloss.Color("#04E762")
	cResmaRed     = lipgloss.Color("#FF5C5C")
	cResmaWarning = lipgloss.Color("#FFB400")
	cResmaWhite   = lipgloss.Color("#FFFFFF")
	cResmaCursor  = lipgloss.Color("#4B0082") // Indigo cursor background
	cResmaBorder  = lipgloss.Color("#3D3D5C")
	cResmaPrimary = lipgloss.Color("#7D56F3") // RESMA Violet
	cResmaTabBg   = lipgloss.Color("#2A2A3E")
)

// Estilos de alta fidelidade
var (
	sHeader = lipgloss.NewStyle().
		Padding(0, 1)

	sClusterTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cResmaAqua)

	sInfoKey = lipgloss.NewStyle().
			Foreground(cResmaGray)

	sInfoVal = lipgloss.NewStyle().
			Bold(true).
			Foreground(cResmaWhite)

	sMenuKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(cResmaCyan)

	sMenuDesc = lipgloss.NewStyle().
			Foreground(cResmaGray)

	sLogo = lipgloss.NewStyle().
		Foreground(cResmaPrimary).
		Bold(true)

	sStatus = lipgloss.NewStyle().
		Bold(true).
		Foreground(cResmaBlack).
		Padding(0, 2).
		AlignHorizontal(lipgloss.Center)

	sTableCursor = lipgloss.NewStyle().
			Background(cResmaCursor).
			Foreground(cResmaWhite).
			Bold(true)

	sTableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(cResmaAqua).
			Underline(true)

	sTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(cResmaWhite).
			Background(cResmaPrimary).
			Padding(0, 2).
			MarginRight(1)

	sTabInactive = lipgloss.NewStyle().
			Foreground(cResmaMuted).
			Background(cResmaTabBg).
			Padding(0, 2).
			MarginRight(1)

	sCrumbActive = lipgloss.NewStyle().
			Bold(true).
			Background(cResmaPrimary).
			Foreground(cResmaWhite)

	sCrumbInactive = lipgloss.NewStyle().
			Background(cResmaBorder).
			Foreground(cResmaGray)

	sFlash = lipgloss.NewStyle().
		Bold(true).
		Foreground(cResmaAqua).
		AlignHorizontal(lipgloss.Center)

	// Semantics
	sSuccess   = lipgloss.NewStyle().Foreground(cResmaGreen).Bold(true)
	sWarning   = lipgloss.NewStyle().Foreground(cResmaWarning).Bold(true)
	sError     = lipgloss.NewStyle().Foreground(cResmaRed).Bold(true)
	sMuted     = lipgloss.NewStyle().Foreground(cResmaMuted)
	sHighlight = lipgloss.NewStyle().Foreground(cResmaPrimary).Bold(true)
)
