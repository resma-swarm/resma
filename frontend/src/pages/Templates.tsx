import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { InputTags } from "@/components/ui/input-tags"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Combobox } from "@/components/combobox"
import { toast } from "sonner"
import { PageHeader } from "@/components/page-header"
import { YamlEditor } from "@/components/yaml-editor"
import { Zap, AlertCircle, Package, Plus, Pencil, Trash2, Save, Cpu, MemoryStick } from "lucide-react"
import { useState } from "react"
import * as yaml from "yaml"

interface Template {
  id: number
  name: string
  description: string
  yaml_content: string
  stacks: string[]
  created_at?: string
  updated_at?: string
}

interface Service {
  name: string
  container_count: number
  cpu_p95: number
  mem_p99: number
  status?: string
  last_seen?: string | null
}

interface ParsedTemplate {
  limits?: { cpus?: string; memory?: string }
  reservations?: { cpus?: string; memory?: string }
  mem_margin?: number
  cpu_margin?: number
  reservation_ratio?: number
  leak_tolerance?: number
}

const DEFAULT_YAML = `# Limites maximos que o container pode consumir
limits:
  cpus: '0.50'     # em cores (ex: 0.50 = meio core)
  memory: 512M     # ex: 512M, 1G, 2G
# Reservas garantidas para o container
reservations:
  cpus: '0.25'
  memory: 256M
# Margens de seguranca sobre P95/P99
mem_margin: 1.5       # multiplicador sobre P99 de memoria (1.5 = +50%)
cpu_margin: 1.5       # multiplicador sobre P95 de CPU (1.5 = +50%)
# Razao reserva/limite (0.75 = reserva = 75% do limite)
reservation_ratio: 0.75
# Tolerancia para deteccao de memory leak
leak_tolerance: 1.0
`

function parseTemplate(yamlContent: string): ParsedTemplate | null {
  try {
    const parsed = yaml.parse(yamlContent)
    return parsed as ParsedTemplate
  } catch {
    return null
  }
}

function ResourceStat({ icon: Icon, label, value, color }: { icon: React.ElementType; label: string; value: string; color: string }) {
  return (
    <div className="flex items-center gap-2 rounded-lg bg-muted/50 px-2.5 py-1.5">
      <Icon className={`h-3.5 w-3.5 ${color}`} />
      <div className="flex flex-col">
        <span className="text-[10px] text-muted-foreground uppercase tracking-wide">{label}</span>
        <span className="text-xs font-medium">{value}</span>
      </div>
    </div>
  )
}

function TemplateCard({ tmpl, onEdit, onDelete }: { tmpl: Template; onEdit: (t: Template) => void; onDelete: (t: Template) => void }) {
  const [activeTab, setActiveTab] = useState<"visual" | "yaml">("visual")
  const parsed = parseTemplate(tmpl.yaml_content)

  const limits = parsed?.limits
  const reservations = parsed?.reservations

  return (
    <Card className="overflow-hidden hover:border-primary/50 hover:shadow-lg hover:-translate-y-0.5 transition-all duration-200">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-chart-5/15 text-chart-5">
              <Package className="h-4.5 w-4.5" />
            </div>
            <div>
              <CardTitle className="text-base">{tmpl.name}</CardTitle>
              <CardDescription className="text-xs mt-0.5">{tmpl.description}</CardDescription>
            </div>
          </div>
          <div className="flex gap-1">
            <Button variant="ghost" size="sm" onClick={() => onEdit(tmpl)}>
              <Pencil className="h-3.5 w-3.5" />
            </Button>
            {tmpl.name !== "default" && (
              <Button variant="ghost" size="sm" onClick={() => onDelete(tmpl)}>
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {(tmpl.stacks?.length ?? 0) > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {(tmpl.stacks ?? []).map((stack) => (
              <Badge key={stack} variant="outline" className="text-xs border-chart-5/40 text-chart-5">
                {stack}
              </Badge>
            ))}
          </div>
        )}

        <div className="flex gap-1 border-b border-border">
          <button
            className={`px-3 py-1.5 text-xs font-medium transition-colors ${
              activeTab === "visual"
                ? "bg-background text-foreground border-b-2 border-primary rounded-b-none -mb-px"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("visual")}
          >
            Visual
          </button>
          <button
            className={`px-3 py-1.5 text-xs font-medium transition-colors ${
              activeTab === "yaml"
                ? "bg-background text-foreground border-b-2 border-primary rounded-b-none -mb-px"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("yaml")}
          >
            YAML
          </button>
        </div>

        {activeTab === "visual" && (
          <>
            {parsed && (
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-1.5">
                  <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Limits</span>
                  <ResourceStat icon={Cpu} label="CPU" value={limits?.cpus ? `${limits.cpus}` : "—"} color="text-primary" />
                  <ResourceStat icon={MemoryStick} label="Mem" value={limits?.memory ?? "—"} color="text-chart-2" />
                </div>
                <div className="space-y-1.5">
                  <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Reservations</span>
                  <ResourceStat icon={Cpu} label="CPU" value={reservations?.cpus ? `${reservations.cpus}` : "—"} color="text-primary" />
                  <ResourceStat icon={MemoryStick} label="Mem" value={reservations?.memory ?? "—"} color="text-chart-2" />
                </div>
              </div>
            )}

            {!parsed && (
              <div className="flex items-center gap-2 rounded-lg bg-destructive/10 px-3 py-2">
                <AlertCircle className="h-4 w-4 text-destructive" />
                <span className="text-xs text-destructive">YAML inválido</span>
              </div>
            )}
          </>
        )}

        {activeTab === "yaml" && (
          <div className="rounded-md border overflow-hidden">
            <YamlEditor
              value={tmpl.yaml_content}
              readOnly
              height="280px"
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function Templates() {
  const queryClient = useQueryClient()
  const [selectedTemplate, setSelectedTemplate] = useState<string>("")
  const [selectedService, setSelectedService] = useState<string>("")
  const [showEditor, setShowEditor] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<Template | null>(null)
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    yaml_content: DEFAULT_YAML,
    stacks: [] as string[],
  })
  const [formError, setFormError] = useState<string | null>(null)

  const { data: templates, isLoading: templatesLoading } = useQuery<Template[]>({
    queryKey: ["templates"],
    queryFn: () => api.get<Template[]>("/templates"),
  })

  const { data: services } = useQuery<Service[]>({
    queryKey: ["services"],
    queryFn: () => api.get<Service[]>("/services"),
  })

  const applyMutation = useMutation({
    mutationFn: ({ template, service }: { template: string; service: string }) =>
      api.post<{ success: boolean; message: string }>(`/templates/${template}/apply/${service}`),
    onSuccess: (data) => {
      if (data.success) {
        toast.success(data.message)
      } else {
        toast.error(data.message)
      }
      queryClient.invalidateQueries({ queryKey: ["services"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Erro ao aplicar template")
    },
  })

  const createMutation = useMutation({
    mutationFn: (body: { name: string; description: string; yaml_content: string; stacks: string[] }) =>
      api.post<{ success: boolean; id: number; message: string }>("/templates", body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["templates"] })
      setShowEditor(false)
      setFormError(null)
      toast.success("Template criado com sucesso")
    },
    onError: (err) => {
      setFormError(err instanceof Error ? err.message : "Erro ao criar template")
      toast.error(err instanceof Error ? err.message : "Erro ao criar template")
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: number; body: { name?: string; description?: string; yaml_content?: string; stacks?: string[] } }) =>
      api.put<{ success: boolean; message: string }>(`/templates/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["templates"] })
      setShowEditor(false)
      setEditingTemplate(null)
      setFormError(null)
      toast.success("Template atualizado com sucesso")
    },
    onError: (err) => {
      setFormError(err instanceof Error ? err.message : "Erro ao atualizar template")
      toast.error(err instanceof Error ? err.message : "Erro ao atualizar template")
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete<{ success: boolean; message: string }>(`/templates/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["templates"] })
      toast.success("Template removido com sucesso")
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Erro ao remover template")
    },
  })

  const handleApply = () => {
    if (!selectedTemplate || !selectedService) return
    applyMutation.mutate({ template: selectedTemplate, service: selectedService })
  }

  const handleNewTemplate = () => {
    setEditingTemplate(null)
    setFormData({ name: "", description: "", yaml_content: DEFAULT_YAML, stacks: [] })
    setFormError(null)
    setShowEditor(true)
  }

  const handleEditTemplate = (tmpl: Template) => {
    setEditingTemplate(tmpl)
    setFormData({
      name: tmpl.name,
      description: tmpl.description,
      yaml_content: tmpl.yaml_content,
      stacks: tmpl.stacks,
    })
    setFormError(null)
    setShowEditor(true)
  }

  const handleSave = () => {
    if (!formData.name.trim()) {
      setFormError("Nome é obrigatório")
      return
    }
    if (editingTemplate) {
      updateMutation.mutate({
        id: editingTemplate.id,
        body: {
          name: formData.name,
          description: formData.description,
          yaml_content: formData.yaml_content,
          stacks: formData.stacks,
        },
      })
    } else {
      createMutation.mutate(formData)
    }
  }

  const handleDelete = (tmpl: Template) => {
    if (!confirm(`Remover template "${tmpl.name}"?`)) return
    deleteMutation.mutate(tmpl.id)
  }

  if (templatesLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Templates" description="Perfis de recursos pré-definidos">
        <div className="flex items-center gap-2">
          <Badge variant="outline">{templates?.length ?? 0} templates</Badge>
          <Button onClick={handleNewTemplate} size="sm">
            <Plus className="h-4 w-4" />
            Novo
          </Button>
        </div>
      </PageHeader>

      <Dialog
        open={showEditor}
        onOpenChange={(open) => {
          setShowEditor(open)
          if (!open) setEditingTemplate(null)
        }}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {editingTemplate ? `Editar: ${editingTemplate.name}` : "Novo Template"}
            </DialogTitle>
            <DialogDescription>Configure recursos e stacks associadas</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="tmpl-name">Nome</Label>
                <Input
                  id="tmpl-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  disabled={!!editingTemplate && editingTemplate.name === "default"}
                  placeholder="ex: postgres, node-api"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="tmpl-desc">Descrição</Label>
                <Input
                  id="tmpl-desc"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Descrição do template"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label>Stacks (labels resma.stack)</Label>
              <InputTags
                value={formData.stacks}
                onChange={(stacks) => setFormData((prev) => ({ ...prev, stacks: typeof stacks === "function" ? stacks(prev.stacks) : stacks }))}
                placeholder="ex: postgres, mysql, node-api"
              />
            </div>

            <div className="space-y-1.5">
              <Label>Configuração YAML</Label>
              <div className="rounded-md border overflow-hidden">
                <YamlEditor
                  value={formData.yaml_content}
                  onChange={(val) => setFormData({ ...formData, yaml_content: val })}
                  height="350px"
                />
              </div>
            </div>

            {formError && (
              <div className="flex items-center gap-2 rounded-lg bg-destructive/10 px-3 py-2">
                <AlertCircle className="h-4 w-4 text-destructive" />
                <span className="text-xs text-destructive">{formError}</span>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowEditor(false); setEditingTemplate(null) }}>
              Cancelar
            </Button>
            <Button onClick={handleSave} disabled={createMutation.isPending || updateMutation.isPending}>
              <Save className="h-4 w-4" />
              {createMutation.isPending || updateMutation.isPending ? "Salvando..." : "Salvar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Aplicar Template</CardTitle>
          <CardDescription>Selecione um template e um serviço para aplicar</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <Combobox
              options={(templates ?? []).map((t) => ({ value: t.name, label: t.name }))}
              value={selectedTemplate}
              onChange={setSelectedTemplate}
              placeholder="Template"
              searchPlaceholder="Buscar template..."
              emptyText="Nenhum template encontrado."
              className="sm:w-48"
            />
            <Combobox
              options={(services ?? []).map((s) => ({
                value: s.name,
                label: s.name,
                icon: (
                  <span
                    className={`h-2 w-2 shrink-0 rounded-full ${
                      s.status === "online" ? "bg-green-500" :
                      s.status === "offline" ? "bg-amber-500" :
                      "bg-muted-foreground"
                    }`}
                  />
                ),
              }))}
              value={selectedService}
              onChange={setSelectedService}
              placeholder="Serviço"
              searchPlaceholder="Buscar serviço..."
              emptyText="Nenhum serviço encontrado."
              className="sm:w-48"
            />
            <Button
              onClick={handleApply}
              disabled={!selectedTemplate || !selectedService || applyMutation.isPending}
              className="sm:ml-auto"
            >
              <Zap className="h-4 w-4" />
              {applyMutation.isPending ? "Aplicando..." : "Aplicar"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {templates?.map((tmpl) => (
          <TemplateCard
            key={tmpl.id}
            tmpl={tmpl}
            onEdit={handleEditTemplate}
            onDelete={handleDelete}
          />
        ))}
      </div>
    </div>
  )
}
