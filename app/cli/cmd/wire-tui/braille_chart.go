package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Braille dot masks (U+2800 base)
// Vertical layout (left column): dot1=top(0x01), dot2=mid(0x02), dot3=bot(0x04)
// Vertical layout (right column): dot4=top(0x08), dot5=mid(0x10), dot6=bot(0x20)
const (
	brailleBase = 0x2800
	dot1        = 0x01 // top-left
	dot2        = 0x02 // mid-left
	dot3        = 0x04 // bot-left
	dot4        = 0x08 // top-right
	dot5        = 0x10 // mid-right
	dot6        = 0x20 // bot-right
)

// levelToDotsLeft mapeia nível 0-3 para dots da coluna esquerda (preenche de baixo p/ cima).
func levelToDotsLeft(level int) int {
	switch level {
	case 1:
		return dot3 // bottom only
	case 2:
		return dot2 | dot3 // bottom + middle
	case 3:
		return dot1 | dot2 | dot3 // full
	default:
		return 0
	}
}

// levelToDotsRight mapeia nível 0-3 para dots da coluna direita.
func levelToDotsRight(level int) int {
	switch level {
	case 1:
		return dot6
	case 2:
		return dot5 | dot6
	case 3:
		return dot4 | dot5 | dot6
	default:
		return 0
	}
}

// brailleSparkline renderiza uma sparkline de 1 linha usando braille Unicode.
// Cada célula comporta 2 data points com 4 níveis verticais cada — 2× mais
// densidade horizontal e 4× mais suave que blocks (▁▂▃▄▅▆▇█).
// Cores com gradiente: cyan (low) → green (mid) → yellow (high) → red (peak).
func brailleSparkline(data []float64, width int) string {
	if len(data) == 0 || width < 1 {
		return strings.Repeat(" ", width)
	}

	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return strings.Repeat(" ", width)
	}

	// Normalizar para níveis 0-3
	levels := make([]int, len(data))
	for i, v := range data {
		levels[i] = int((v / maxVal) * 3.99)
		if levels[i] > 3 {
			levels[i] = 3
		}
	}

	// Construir chars braille (2 data points por célula)
	var runes []rune
	for i := 0; i < len(levels); i += 2 {
		offset := levelToDotsLeft(levels[i])
		if i+1 < len(levels) {
			offset |= levelToDotsRight(levels[i+1])
		}
		runes = append(runes, rune(brailleBase+offset))
	}

	// Truncar ou pad para width
	result := string(runes)
	r := []rune(result)
	if len(r) > width {
		r = r[:width]
	} else if len(r) < width {
		// pad com braille vazio (⠀) à direita
		r = append(r, []rune(strings.Repeat(string(rune(brailleBase)), width-len(r)))...)
	}

	// Gradiente de cor baseado na média dos valores
	avg := 0.0
	for _, v := range data {
		avg += v
	}
	avg /= float64(len(data))
	ratio := avg / maxVal

	var color lipgloss.Color
	switch {
	case ratio > 0.75:
		color = cResmaRed
	case ratio > 0.50:
		color = cResmaWarning
	case ratio > 0.25:
		color = cResmaGreen
	default:
		color = cResmaCyan
	}

	return lipgloss.NewStyle().Foreground(color).Render(string(r))
}

// brailleSparklinePlain é a versão sem cores (para linhas selecionadas).
func brailleSparklinePlain(data []float64, width int) string {
	if len(data) == 0 || width < 1 {
		return strings.Repeat(" ", width)
	}

	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return strings.Repeat(" ", width)
	}

	levels := make([]int, len(data))
	for i, v := range data {
		levels[i] = int((v / maxVal) * 3.99)
		if levels[i] > 3 {
			levels[i] = 3
		}
	}

	var runes []rune
	for i := 0; i < len(levels); i += 2 {
		offset := levelToDotsLeft(levels[i])
		if i+1 < len(levels) {
			offset |= levelToDotsRight(levels[i+1])
		}
		runes = append(runes, rune(brailleBase+offset))
	}

	result := string(runes)
	r := []rune(result)
	if len(r) > width {
		r = r[:width]
	} else if len(r) < width {
		r = append(r, []rune(strings.Repeat(string(rune(brailleBase)), width-len(r)))...)
	}

	return string(r)
}

// brailleChart renderiza um chart multi-linha (2+ rows) usando braille.
// Cada célula tem 2×4 = 8 pixels virtuais, dando linhas suaves.
// Usado em detail views para visualização rica de métricas.
func brailleChart(data []float64, width, height int) string {
	if len(data) == 0 || width < 1 || height < 1 {
		return strings.Repeat("\n", height)
	}

	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return strings.Repeat("\n", height)
	}

	// Grid braille: width cells × height cells = 2*width × 4*height pixels virtuais
	// Cada célula braille tem 2 colunas × 4 linhas de pixels
	pixelW := width * 2
	pixelH := height * 4

	// Mapear data points para pixels X
	// Resample: distribuir data points ao longo de pixelW
	pixels := make([][]bool, pixelH) // [y][x]
	for y := range pixels {
		pixels[y] = make([]bool, pixelW)
	}

	for i, v := range data {
		x := (i * pixelW) / len(data)
		if x >= pixelW {
			x = pixelW - 1
		}
		// Normalizar valor para pixelY (0 = topo, pixelH-1 = base)
		normalized := v / maxVal
		y := pixelH - 1 - int(normalized*float64(pixelH-1))
		if y < 0 {
			y = 0
		}
		if y >= pixelH {
			y = pixelH - 1
		}
		// Preencher do base até y (area fill suave)
		for fillY := pixelH - 1; fillY >= y; fillY-- {
			pixels[fillY][x] = true
		}
	}

	// Interpolar pixels entre data points para linha contínua
	for x := 1; x < pixelW; x++ {
		// encontrar y mais alto ativo
		topY := -1
		for y := 0; y < pixelH; y++ {
			if pixels[y][x] {
				topY = y
				break
			}
		}
		if topY == -1 {
			continue
		}
		// encontrar y do pixel anterior
		prevTopY := -1
		for y := 0; y < pixelH; y++ {
			if pixels[y][x-1] {
				prevTopY = y
				break
			}
		}
		if prevTopY == -1 {
			continue
		}
		// interpolar linha entre prevTopY e topY
		if prevTopY > topY {
			prevTopY, topY = topY, prevTopY
		}
		for interY := prevTopY; interY <= topY; interY++ {
			pixels[interY][x-1] = true
		}
	}

	// Renderizar grid braille
	var sb strings.Builder
	for cellY := 0; cellY < height; cellY++ {
		for cellX := 0; cellX < width; cellX++ {
			offset := 0
			// 4 linhas de pixels por célula (top=0, bot=3)
			for py := 0; py < 4; py++ {
				pixelY := cellY*4 + py
				if pixelY >= pixelH {
					continue
				}
				// coluna esquerda (px=0)
				pxLeft := cellX * 2
				if pxLeft < pixelW && pixels[pixelY][pxLeft] {
					switch py {
					case 0:
						offset |= dot1
					case 1:
						offset |= dot2
					case 2:
						offset |= dot3
					}
				}
				// coluna direita (px=1)
				pxRight := cellX*2 + 1
				if pxRight < pixelW && pixels[pixelY][pxRight] {
					switch py {
					case 0:
						offset |= dot4
					case 1:
						offset |= dot5
					case 2:
						offset |= dot6
					}
				}
			}
			sb.WriteRune(rune(brailleBase + offset))
		}
		if cellY < height-1 {
			sb.WriteString("\n")
		}
	}

	// Gradiente de cor
	avg := 0.0
	for _, v := range data {
		avg += v
	}
	avg /= float64(len(data))
	ratio := avg / maxVal

	var color lipgloss.Color
	switch {
	case ratio > 0.75:
		color = cResmaRed
	case ratio > 0.50:
		color = cResmaWarning
	case ratio > 0.25:
		color = cResmaGreen
	default:
		color = cResmaCyan
	}

	return lipgloss.NewStyle().Foreground(color).Render(sb.String())
}
