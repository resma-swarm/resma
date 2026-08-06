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
  { name: "template", label: "Template", description: "Perfil YAML pré-definido — aplicação manual sem ML" },
]

function formatCpu(cores: number): string {
  if (cores >= 1) return `${cores.toFixed(1)}c`
  return `${cores.toFixed(2)}c`
}

export function LayerToggle({ value, onChange, suggestedTiers }: LayerToggleProps) {
  return (
    <div className="flex items-center gap-2">
      <ToggleGroup
        type="single"
        value={value}
        onValueChange={(v) => {
          if (v) onChange(v as TierName)
        }}
        className="gap-0 rounded-lg border bg-muted/30 p-0.5"
      >
        {tierConfig.map((tier) => {
          const t = tier.name === "template" ? null : suggestedTiers[tier.name]
          return (
            <ToggleGroupItem
              key={tier.name}
              value={tier.name}
              variant="outline"
              className="flex flex-col items-center gap-0.5 px-3 py-1.5 h-auto rounded-md border-0 data-[state=on]:bg-primary/10 data-[state=on]:shadow-sm"
              aria-label={tier.label}
            >
              <span className="text-xs font-medium">{tier.label}</span>
              <span className="text-[10px] text-muted-foreground tabular-nums">
                {tier.name === "template" ? "YAML" : (
                  <>
                    {t && t.cpu_limit > 0 ? formatCpu(t.cpu_limit) : "—"}
                    {" · "}
                    {t && t.mem_limit > 0 ? formatBytes(t.mem_limit) : "—"}
                  </>
                )}
              </span>
            </ToggleGroupItem>
          )
        })}
      </ToggleGroup>
      <HelpIcon
        title="Camadas de recomendação"
        text="Conservadora: margem maior, mais seguro. Equilibrada: data-driven, recomendada. Agressiva: margem mínima, requer rollback ativo. Template: perfil YAML pré-definido manual."
        side="top"
        className="h-4 w-4 text-muted-foreground"
      />
    </div>
  )
}
