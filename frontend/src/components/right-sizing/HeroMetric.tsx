/**
 * HeroMetric — card no topo da página mostrando recursos liberados.
 *
 * Spec: hero-metric.md §3
 * shadcn: Card, CardContent, Separator, Badge, Skeleton
 * Cálculo: client-side via calculateHero() (spec ml-payload-schema.md §9)
 */
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Cpu, MemoryStick, TrendingUp } from "lucide-react"
import { formatBytes } from "@/lib/utils"
import type { HeroData } from "./types"

interface HeroMetricProps {
  data: HeroData | null
  loading: boolean
}

export function HeroMetric({ data, loading }: HeroMetricProps) {
  if (loading || !data) {
    return (
      <Card className="mb-4">
        <CardContent className="flex items-center gap-6 p-4">
          <Skeleton className="h-16 w-32" />
          <Skeleton className="h-12 w-px" />
          <Skeleton className="h-16 w-32" />
          <Skeleton className="h-12 w-px" />
          <Skeleton className="h-16 w-32" />
        </CardContent>
      </Card>
    )
  }

  const hasPotential = data.cpu_cores > 0 || data.mem_bytes > 0

  return (
    <Card className="mb-4">
      <CardContent className="flex items-center gap-6 p-4 flex-wrap">
        <div className="flex items-center gap-3">
          <TrendingUp className="h-8 w-8 text-primary" />
          <div>
            <p className="text-xs text-muted-foreground">Pode liberar</p>
            <p className="text-2xl font-bold tabular-nums">
              {data.pending_count}
              <span className="text-sm font-normal text-muted-foreground ml-1">serviços</span>
            </p>
          </div>
        </div>

        <Separator orientation="vertical" className="h-12" />

        <div className="flex items-center gap-3">
          <Cpu className="h-8 w-8 text-chart-2" />
          <div>
            <p className="text-xs text-muted-foreground">CPU liberada</p>
            <p className="text-2xl font-bold tabular-nums">
              {data.cpu_cores.toFixed(1)}
              <span className="text-sm font-normal text-muted-foreground ml-1">cores</span>
            </p>
          </div>
        </div>

        <Separator orientation="vertical" className="h-12" />

        <div className="flex items-center gap-3">
          <MemoryStick className="h-8 w-8 text-chart-3" />
          <div>
            <p className="text-xs text-muted-foreground">RAM liberada</p>
            <p className="text-2xl font-bold tabular-nums">{formatBytes(data.mem_bytes)}</p>
          </div>
        </div>

        <div className="ml-auto flex flex-col items-end gap-1">
          {data.optimized_count > 0 ? (
            <>
              <Badge variant="success">
                {data.optimized_count} otimizados
              </Badge>
              {data.delta_optimized_count > 0 && (
                <span className="text-xs text-muted-foreground">
                  +{data.delta_optimized_count} vs mês anterior
                </span>
              )}
            </>
          ) : hasPotential ? (
            <Badge variant="secondary">{data.pending_count} pendentes de revisão</Badge>
          ) : (
            <Badge variant="outline">Tudo otimizado</Badge>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
