// Package tui — FlashField: componente reutilizável de "valor que acabou de mudar".
//
// FlashField rastreia um valor e um timestamp. Quando o valor muda, registra
// o momento da mudança. Enquanto time.Since(flashAt) < duration, o valor é
// considerado "flashing" e deve ser renderizado com destaque.
//
// Uso típico:
//
//	type model struct {
//	    cpuFlash FlashField
//	    memFlash FlashField
//	}
//
//	// Ao receber novo valor via SSE:
//	m.cpuFlash.Update(newCPUValue)
//
//	// Ao renderizar:
//	if m.cpuFlash.Flashing() {
//	    renderFlash(m.cpuFlash.Value)
//	} else {
//	    renderNormal(m.cpuFlash.Value)
//	}
package tui

import "time"

// FlashField rastreia um valor numérico e seu timestamp de última mudança.
// É genérico o suficiente para qualquer tipo de dado que muda via SSE
// (CPU, MEM, contagem de containers, alerts, etc).
type FlashField struct {
	value    float64 // último valor recebido
	hasValue bool    // se já recebeu algum valor (diferencia 0 real de "sem dados")
	flashAt  time.Time
	duration time.Duration
}

// NewFlashField cria um FlashField com a duração do efeito de flash.
// Recomendado: 600-1000ms (curto o suficiente para não incomodar).
func NewFlashField(duration time.Duration) FlashField {
	return FlashField{duration: duration}
}

// Update registra um novo valor. Se o valor for diferente do atual,
// aciona o flash. Se for igual, não faz nada (evita flash desnecessário).
// O primeiro valor (hasValue == false) sempre aciona o flash.
func (f *FlashField) Update(v float64) {
	if !f.hasValue || f.value != v {
		f.value = v
		f.flashAt = time.Now()
	}
	f.hasValue = true
}

// Value retorna o valor atual e se já há um valor válido.
func (f *FlashField) Value() (float64, bool) {
	return f.value, f.hasValue
}

// Flashing retorna true se o valor está dentro da janela de flash
// (acabou de mudar e ainda não expirou).
func (f FlashField) Flashing() bool {
	return f.hasValue && time.Since(f.flashAt) < f.duration
}

// HasValue retorna true se já recebeu algum valor via SSE.
func (f FlashField) HasValue() bool {
	return f.hasValue
}
