import { useState, useEffect } from "react"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { HelpIcon } from "@/components/help-icon"
import { Save } from "lucide-react"
import { toast } from "sonner"

interface Settings {
  collect_interval: number
  retention_days: number
  outlier_threshold: number
  leak_r2_threshold: number
  leak_daily_mb_threshold: number
  analysis_window_days: number
  cluster_interval: number
  storage_interval: number
  stale_service_days: number
}

const SETTING_META: { key: keyof Settings; label: string; description: string; unit?: string; min: number; max: number }[] = [
  { key: "collect_interval", label: "Intervalo de Coleta", description: "Segundos entre cada coleta de métricas dos containers. Ex: 15s significa que a cada 15 segundos o agent lê CPU/memória de todos os containers. Valores baixos = mais detalhe, mais armazenamento. Valores altos = menos overhead, menos resolução.", unit: "segundos", min: 5, max: 3600 },
  { key: "retention_days", label: "Retenção de Dados", description: "Dias de métricas mantidos no DuckDB antes da purga automática. Ex: 30d mantém 1 mês de histórico. Dados mais antigos são deletados para controlar o tamanho do banco.", unit: "dias", min: 1, max: 3650 },
  { key: "outlier_threshold", label: "Threshold de Outlier", description: "Número de desvios-padrão (σ) para detectar valores atípicos. Ex: 3σ descarta leituras que estejam a mais de 3 desvios da média (ex: pico de CPU por um processo pontual). Protege as estatísticas P95/P99 de distorções.", unit: "σ", min: 0.1, max: 10 },
  { key: "leak_r2_threshold", label: "R² para Leak Detection", description: "Coeficiente de determinação (R²) mínimo para considerar crescimento de memória como leak. Ex: 0.85 significa que o uso de memória precisa ter 85% de correlação linear com o tempo para ser classificado como leak. Quanto mais próximo de 1, mais confiável a detecção.", unit: "R²", min: 0, max: 1 },
  { key: "leak_daily_mb_threshold", label: "Crescimento Diário (MB)", description: "Crescimento mínimo em MB por dia para classificar como memory leak. Ex: 10 MB/dia significa que um serviço que cresce 10+ MB/dia de forma consistente será alertado como leak. Evita falsos positivos com serviços que crescem pouco.", unit: "MB/dia", min: 0, max: 10000 },
  { key: "analysis_window_days", label: "Janela de Análise", description: "Dias de dados usados pela análise de ML para gerar recomendações de limites. Ex: 7d usa a última semana para calcular P95/P99 e sugerir CPU/memória. Janelas maiores = recomendações mais estáveis, janelas menores = mais reativas a mudanças recentes.", unit: "dias", min: 1, max: 365 },
  { key: "cluster_interval", label: "Intervalo de Cluster", description: "Segundos entre coletas de informações do cluster Docker Swarm (nodes, managers, quorum). Ex: 30s verifica a cada 30 segundos se nodes estão up/down. Não precisa ser tão frequente quanto a coleta de containers.", unit: "segundos", min: 5, max: 3600 },
  { key: "storage_interval", label: "Intervalo de Storage", description: "Segundos entre coletas de métricas de storage (volumes e discos). Ex: 60s lê df/du a cada minuto. Storage muda devagar, então intervalos maiores são aceitáveis e reduzem I/O no disco.", unit: "segundos", min: 5, max: 3600 },
  { key: "stale_service_days", label: "Stale Service Days", description: "Dias sem heartbeat para marcar um serviço como stale (obsoleto). Ex: 7d significa que serviços sem coleta há 7+ dias são marcados como stale e saem dos cálculos ativos. Útil para serviços que foram removidos do Swarm mas ainda constam no banco.", unit: "dias", min: 1, max: 365 },
]

// isInvalid — verifica se um valor está fora do range permitido.
// Retorna true se inválido (bloqueia Save e marca visualmente o input).
function isInvalid(value: number, min: number, max: number): boolean {
  return isNaN(value) || value < min || value > max
}

export function ParametersPage() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [original, setOriginal] = useState<Settings | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.get<Settings>("/settings").then((data) => {
      setSettings(data)
      setOriginal(data)
    }).catch((e) => toast.error("Erro ao carregar: " + (e as Error).message))
  }, [])

  const hasChanges = () => {
    if (!settings || !original) return false
    return SETTING_META.some((m) => settings[m.key] !== original[m.key])
  }

  // hasInvalid — bloqueia Save se qualquer valor estiver fora do range.
  // Feedback inline: borda vermelha + mensagem abaixo do input inválido.
  const hasInvalid = () => {
    if (!settings) return false
    return SETTING_META.some((m) => isInvalid(settings[m.key], m.min, m.max))
  }

  const handleSave = async () => {
    if (!settings || !original) return
    const changes: Record<string, number> = {}
    for (const m of SETTING_META) {
      if (settings[m.key] !== original[m.key]) {
        changes[m.key] = settings[m.key]
      }
    }
    if (Object.keys(changes).length === 0) return
    setSaving(true)
    try {
      await api.put("/settings", changes)
      toast.success("Parâmetros salvos")
      setOriginal({ ...settings })
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  if (!settings) return <div className="text-muted-foreground">Carregando...</div>

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Parâmetros Operacionais</CardTitle>
          <CardDescription>
            Valores persistidos no banco (DB). Env vars permanecem como defaults/infra.
            Mudanças não exigem restart do container.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {SETTING_META.map((m) => {
              const invalid = isInvalid(settings[m.key], m.min, m.max)
              return (
                <div key={m.key} className="space-y-2">
                  <div className="flex items-center gap-1.5">
                    <Label htmlFor={m.key}>
                      {m.label}
                      {m.unit && <span className="ml-1 text-muted-foreground">({m.unit})</span>}
                    </Label>
                    <HelpIcon text={m.description} side="top" />
                  </div>
                  <Input
                    id={m.key}
                    type="number"
                    step={m.key.includes("threshold") ? "0.1" : "1"}
                    min={m.min}
                    max={m.max}
                    value={settings[m.key]}
                    onChange={(e) => {
                      const val = m.key.includes("threshold")
                        ? parseFloat(e.target.value)
                        : parseInt(e.target.value)
                      setSettings({ ...settings, [m.key]: isNaN(val) ? 0 : val })
                    }}
                    className={invalid ? "border-destructive focus-visible:ring-destructive" : ""}
                  />
                  {invalid && (
                    <p className="text-xs text-destructive">
                      Valor deve estar entre {m.min} e {m.max}
                    </p>
                  )}
                </div>
              )
            })}
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              variant="outline"
              onClick={() => setSettings(original)}
              disabled={!hasChanges() || saving}
            >
              Cancelar
            </Button>
            <Button onClick={handleSave} disabled={!hasChanges() || hasInvalid() || saving}>
              <Save className="h-4 w-4" />
              {saving ? "Salvando..." : "Salvar"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
