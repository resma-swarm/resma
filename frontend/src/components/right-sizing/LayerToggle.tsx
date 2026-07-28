/**
 * LayerToggle — segmented control para alternar entre 3 tiers:
 * Conservadora / Equilibrada / Agressiva.
 *
 * Spec: react-components.md §5
 * shadcn: ToggleGroup + ToggleGroupItem
 * Reusa: HelpIcon
 */
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { HelpIcon } from "@/components/help-icon"
import { formatBytes } from "@/lib/utils"
import type { SuggestedTiers, TierName } from "./types"

interface LayerToggleProps {
  value: TierName
  onChange: (tier: TierName) => void
  suggestedTiers: SuggestedTiers
}

const tierConfig: { name: TierName; label: string; description: string }[] = [
  { name: "conservative", label: "Conservadora", description: "2x P95 / 1.8x P99 — produção crítica sem monitoramento pós-apply" },
  { name: "balanced", label: "Equilibrada", description: "Margem data-driven — recomendado para a maioria dos casos" },
  { name: "aggressive", label: "Agressiva", description: "1.1x P95 / 1.1x P99 — dev/staging com rollback ativo" },
]

export function LayerToggle({ value, onChange, suggestedTiers }: LayerToggleProps) {
  return (
    <div className="flex items-center gap-2">
      <ToggleGroup
        type="single"
        value={value}
        onValueChange={(v) => {
          if (v) onChange(v as TierName)
        }}
        className="gap-1"
      >
        {tierConfig.map((tier) => {
          const t = suggestedTiers[tier.name]
          const freed = t.resources_freed
          return (
            <ToggleGroupItem
              key={tier.name}
              value={tier.name}
              variant="outline"
              className="flex flex-col items-center gap-0.5 px-3 py-1.5 h-auto data-[state=on]:bg-primary/10 data-[state=on]:border-primary"
              aria-label={tier.label}
            >
              <span className="text-xs font-medium">{tier.label}</span>
              <span className="text-[10px] text-muted-foreground tabular-nums">
                {freed.cpu_cores > 0 ? `${freed.cpu_cores.toFixed(1)}c` : "—"}
                {" · "}
                {freed.mem_bytes > 0 ? formatBytes(freed.mem_bytes) : "—"}
              </span>
            </ToggleGroupItem>
          )
        })}
      </ToggleGroup>
      <HelpIcon
        title="Camadas de recomendação"
        text="Conservadora: margem maior, mais seguro. Equilibrada: data-driven, recomendada. Agressiva: margem mínima, requer rollback ativo."
        side="top"
        className="h-4 w-4 text-muted-foreground"
      />
    </div>
  )
}
