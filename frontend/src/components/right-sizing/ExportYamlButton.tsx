/**
 * ExportYamlButton — botão para exportar plano YAML declarativo (GitOps-safe).
 *
 * Spec: export-yaml.md §5
 * Chama GET /api/recommendations/export-yaml e faz download do blob.
 */
import { Button } from "@/components/ui/button"
import { Download } from "lucide-react"
import { useState } from "react"
import { api } from "@/api/client"
import { toast } from "sonner"

interface ExportYamlButtonProps {
  services: string[]
  tier: string
  disabled?: boolean
}

export function ExportYamlButton({ services, tier, disabled }: ExportYamlButtonProps) {
  const [loading, setLoading] = useState(false)

  const handleExport = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (services.length > 0) params.set("services", services.join(","))
      params.set("tier", tier)
      const blob = await api.blob(`/recommendations/export-yaml?${params}`)
      if (blob.size === 0) {
        toast.info("Nenhum serviço over-provisioned para exportar")
        return
      }
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = "resma-right-sizing-plan.yaml"
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      toast.success("Plano YAML exportado")
    } catch (err) {
      toast.error(`Erro ao exportar: ${(err as Error).message}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Button variant="outline" size="sm" onClick={handleExport} disabled={disabled || loading}>
      <Download className="mr-2 h-4 w-4" />
      {loading ? "Exportando..." : "Exportar YAML"}
    </Button>
  )
}
