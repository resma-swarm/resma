/**
 * ResourceSlider — sliders what-if para CPU e memória.
 *
 * Spec: react-components.md §6
 * Permite ao usuário arrastar e ver o impacto em tempo real no WhatIfPanel.
 * NÃO aplica mudanças — é apenas simulação.
 *
 * shadcn: Slider (2 instâncias), Label, Input, Button
 * Reusa: HelpIcon
 */
import { Slider } from "@/components/ui/slider"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { HelpIcon } from "@/components/help-icon"
import { Cpu, MemoryStick, Check } from "lucide-react"
import { formatBytes } from "@/lib/utils"

interface ResourceSliderProps {
  cpuCores: number
  memBytes: number
  cpuMin: number
  cpuMax: number
  memMin: number
  memMax: number
  cpuCurrent: number
  memCurrent: number
  cpuSuggested: number
  memSuggested: number
  onCpuChange: (cores: number) => void
  onMemChange: (bytes: number) => void
}

function formatMem(bytes: number): string {
  if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)}GB`
  if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(0)}MB`
  return formatBytes(bytes)
}

export function ResourceSlider({
  cpuCores, memBytes,
  cpuMin, cpuMax,
  memMin, memMax,
  cpuCurrent, memCurrent,
  cpuSuggested, memSuggested,
  onCpuChange, onMemChange,
}: ResourceSliderProps) {
  const cpuStep = 0.05
  const memStepMb = 16
  const memMinMb = Math.round((memMin || 0) / 1e6)
  const memMaxMb = Math.max(memMinMb + 16, Math.round((memMax || 1e9) / 1e6))
  const memValueMb = Math.round((memBytes || 0) / 1e6)
  const cpuCoresSafe = cpuCores || 0
  const cpuMinSafe = cpuMin || 0
  const cpuMaxSafe = Math.max(cpuMinSafe + 0.1, cpuMax || 8)

  const hasSuggested = cpuSuggested > 0 || memSuggested > 0

  const applySuggested = () => {
    if (cpuSuggested > 0) onCpuChange(cpuSuggested)
    if (memSuggested > 0) onMemChange(memSuggested)
  }

  const handleCpuInput = (val: string) => {
    const n = parseFloat(val)
    if (!isNaN(n) && n >= 0) {
      onCpuChange(n)
    }
  }

  const handleMemInput = (val: string) => {
    const n = parseFloat(val)
    if (!isNaN(n) && n >= 0) {
      // Input em MB, converte para bytes
      onMemChange(n * 1e6)
    }
  }

  return (
    <div className="space-y-3 rounded-lg border p-3 bg-muted/30">
      <div className="flex items-center gap-1.5">
        <span className="text-xs font-medium text-muted-foreground">Simulação</span>
        <HelpIcon
          text="Arraste os sliders ou digite o valor exato nos campos numéricos. Nenhuma mudança é aplicada — é apenas uma prévia do impacto."
          side="top"
          className="h-3.5 w-3.5 text-muted-foreground"
        />
        {hasSuggested && (
          <Button
            variant="outline"
            size="sm"
            className="h-6 text-[10px] ml-auto gap-1"
            onClick={applySuggested}
          >
            <Check className="h-3 w-3" />
            Usar sugerido
          </Button>
        )}
      </div>

      {/* CPU Slider + Input */}
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Cpu className="h-3.5 w-3.5 text-chart-2" />
            <Label className="text-xs">CPU (cores)</Label>
          </div>
          <Input
            type="number"
            value={cpuCoresSafe.toFixed(2)}
            onChange={(e) => handleCpuInput(e.target.value)}
            min={cpuMinSafe}
            max={cpuMaxSafe}
            step={cpuStep}
            className="h-7 w-20 text-xs tabular-nums text-right"
          />
        </div>
        <Slider
          value={[Math.max(cpuMinSafe, Math.min(cpuMaxSafe, cpuCoresSafe))]}
          min={cpuMinSafe}
          max={cpuMaxSafe}
          step={cpuStep}
          onValueChange={(v) => onCpuChange(v[0])}
          className="w-full"
        />
        <div className="flex items-center justify-between text-[10px] text-muted-foreground">
          <span>Min: {cpuMinSafe.toFixed(1)}</span>
          {cpuCurrent > 0 && <span className="text-chart-5">Atual: {cpuCurrent.toFixed(2)}</span>}
          {cpuSuggested > 0 && <span className="text-primary">Sug: {cpuSuggested.toFixed(2)}</span>}
          <span>Max: {cpuMaxSafe.toFixed(1)}</span>
        </div>
      </div>

      {/* Memória Slider + Input */}
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <MemoryStick className="h-3.5 w-3.5 text-chart-3" />
            <Label className="text-xs">Memória</Label>
          </div>
          <div className="flex items-center gap-1">
            <Input
              type="number"
              value={memValueMb}
              onChange={(e) => handleMemInput(e.target.value)}
              min={memMinMb}
              max={memMaxMb}
              step={memStepMb}
              className="h-7 w-20 text-xs tabular-nums text-right"
            />
            <span className="text-[10px] text-muted-foreground">MB</span>
          </div>
        </div>
        <Slider
          value={[Math.max(memMinMb, Math.min(memMaxMb, memValueMb))]}
          min={memMinMb}
          max={memMaxMb}
          step={memStepMb}
          onValueChange={(v) => onMemChange(v[0] * 1e6)}
          className="w-full"
        />
        <div className="flex items-center justify-between text-[10px] text-muted-foreground">
          <span>Min: {formatMem(memMin || 0)}</span>
          {memCurrent > 0 && <span className="text-chart-5">Atual: {formatMem(memCurrent)}</span>}
          {memSuggested > 0 && <span className="text-primary">Sug: {formatMem(memSuggested)}</span>}
          <span>Max: {formatMem(memMax || 1e9)}</span>
        </div>
      </div>
    </div>
  )
}
