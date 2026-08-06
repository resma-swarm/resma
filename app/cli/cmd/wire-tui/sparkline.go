package main

import "strings"

// sparkline gera uma sparkline braille de 1 linha (2× densidade horizontal, 4× vertical vs blocks).
// Cores com gradiente: cyan → green → yellow → red.
func sparkline(data []float64, width int) string {
	return brailleSparkline(data, width)
}

// sparklinePlain é a versão sem cores (para linhas selecionadas).
func sparklinePlain(data []float64) string {
	return brailleSparklinePlain(data, len(data)/2+1)
}

// truncate trunca string para width com ellipsis.
func truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	return s[:width-1] + "…"
}

// padRight preenche string com espaços à direita até width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeft preenche string com espaços à esquerda até width.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
