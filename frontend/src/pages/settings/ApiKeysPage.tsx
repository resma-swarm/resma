import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { Plus, Trash2, Copy, AlertTriangle, Check, KeyRound } from "lucide-react"
import { toast } from "sonner"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface APIKey {
  id: number
  name: string
  prefix: string
  scopes: string
  created_at: string
  last_used_at: string | null
  revoked_at: string | null
}

// timeAgo — timestamp relativo ("há 5min"), padrão do Schedules/Alerts/RollbackWatches.
function timeAgo(ts: string | null): string {
  if (!ts) return "Nunca"
  try {
    const d = new Date(ts)
    const diffMs = Date.now() - d.getTime()
    if (diffMs < 0) return formatTimestamp(ts)
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return "agora"
    if (diffMin < 60) return `há ${diffMin}min`
    const diffH = Math.floor(diffMin / 60)
    if (diffH < 24) return `há ${diffH}h`
    const diffD = Math.floor(diffH / 24)
    return `há ${diffD}d`
  } catch {
    return ts
  }
}

// formatTimestamp — absoluto pt-BR para tooltip (padrão Schedules/Alerts).
function formatTimestamp(ts: string): string {
  if (!ts) return "—"
  try {
    return new Date(ts).toLocaleString("pt-BR", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
  } catch {
    return ts
  }
}

// LastUsedCell — "Nunca" em muted, ou relativo + tooltip absoluto.
function LastUsedCell({ ts }: { ts: string | null }) {
  if (!ts) return <span className="text-muted-foreground">Nunca</span>
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="cursor-help text-xs text-muted-foreground">{timeAgo(ts)}</span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{formatTimestamp(ts)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export function ApiKeysPage() {
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [newKey, setNewKey] = useState({ name: "", scopes: "read" })
  const [revealedKey, setRevealedKey] = useState<string | null>(null)

  const { data: keys, isLoading } = useQuery<APIKey[]>({
    queryKey: ["api-keys"],
    queryFn: () => api.get<APIKey[]>("/auth/api-keys"),
  })
  const keyList = Array.isArray(keys) ? keys : []

  const createMutation = useMutation({
    mutationFn: (payload: typeof newKey) =>
      api.post<{ key: string }>("/auth/api-keys", payload),
    onSuccess: (data) => {
      setRevealedKey(data.key)
      setCreateOpen(false)
      setNewKey({ name: "", scopes: "read" })
      queryClient.invalidateQueries({ queryKey: ["api-keys"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const revokeMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/auth/api-keys/${id}`),
    onSuccess: () => {
      toast.success("API key revogada")
      queryClient.invalidateQueries({ queryKey: ["api-keys"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success("Copiado para área de transferência")
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-8 w-28" />
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nome</TableHead>
                <TableHead>Prefix</TableHead>
                <TableHead>Scopes</TableHead>
                <TableHead>Criada</TableHead>
                <TableHead>Último uso</TableHead>
                <TableHead className="w-[100px]">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-8 w-8" /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{keyList.length} API key(s) ativa(s)</p>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="h-4 w-4" />
              Criar API Key
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Criar API Key</DialogTitle>
              <DialogDescription>
                A chave será mostrada apenas uma vez. Copie e guarde em local seguro.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">Nome</Label>
                <Input
                  id="name"
                  placeholder="ex: production-monitoring"
                  value={newKey.name}
                  onChange={(e) => setNewKey({ ...newKey, name: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label>Scopes</Label>
                <Select value={newKey.scopes} onValueChange={(v) => setNewKey({ ...newKey, scopes: v })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="read">read</SelectItem>
                    <SelectItem value="read,write">read,write</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)}>Cancelar</Button>
              <Button onClick={() => createMutation.mutate(newKey)} disabled={!newKey.name || createMutation.isPending}>
                {createMutation.isPending ? "Criando..." : "Criar"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {revealedKey && (
        <Alert variant="warning">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>API Key criada — copie agora!</AlertTitle>
          <AlertDescription>
            <p className="mb-2">Esta chave não será mostrada novamente.</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded bg-muted p-2 text-xs break-all font-mono">
                {revealedKey}
              </code>
              <Button size="sm" variant="outline" onClick={() => copyToClipboard(revealedKey)}>
                <Copy className="h-4 w-4" />
              </Button>
            </div>
            <Button className="mt-2" size="sm" onClick={() => setRevealedKey(null)}>
              <Check className="h-4 w-4" />
              Concluir
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {keyList.length === 0 ? (
        <div className="rounded-md border">
          <EmptyState
            icon={KeyRound}
            message="Nenhuma API key ativa"
            action={
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="h-4 w-4" />
                Criar API Key
              </Button>
            }
          />
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nome</TableHead>
                <TableHead>Prefix</TableHead>
                <TableHead>Scopes</TableHead>
                <TableHead>Criada</TableHead>
                <TableHead>Último uso</TableHead>
                <TableHead className="w-[100px]">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keyList.map((k) => (
                <TableRow key={k.id}>
                  <TableCell className="font-medium">{k.name}</TableCell>
                  <TableCell><code className="text-xs">{k.prefix}...</code></TableCell>
                  <TableCell>{k.scopes}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(k.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <LastUsedCell ts={k.last_used_at} />
                  </TableCell>
                  <TableCell>
                    {!k.revoked_at && (
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Revogar API key "{k.name}"?</AlertDialogTitle>
                            <AlertDialogDescription>
                              Esta ação é irreversível. Aplicações que usam esta key perderão acesso imediatamente.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancelar</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => revokeMutation.mutate(k.id)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              Revogar
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
