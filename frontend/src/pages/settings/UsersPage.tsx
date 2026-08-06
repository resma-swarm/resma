import { useState, useMemo } from "react"
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
import { Plus, Trash2, ShieldCheck, Users as UsersIcon, Search, ChevronLeft, ChevronRight } from "lucide-react"

const PAGE_SIZE = 20
import { toast } from "sonner"
import { usePermissions } from "@/hooks/use-permissions"

interface User {
  id: number
  username: string
  role: string
  name?: string
}

export function UsersPage() {
  const perms = usePermissions()
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: "", password: "", role: "user", name: "" })
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(0)

  const { data, isLoading } = useQuery<{ users: User[] }>({
    queryKey: ["users"],
    queryFn: () => api.get<{ users: User[] }>("/users"),
  })
  const allUsers = data?.users || []

  // Busca textual — filtra por username e name (case-insensitive).
  const filteredUsers = useMemo(() => {
    if (!search) return allUsers
    const q = search.toLowerCase()
    return allUsers.filter((u) =>
      u.username.toLowerCase().includes(q) ||
      (u.name || "").toLowerCase().includes(q)
    )
  }, [allUsers, search])

  // Paginação client-side (PAGE_SIZE=20) — reset automático quando busca muda.
  const totalPages = Math.max(1, Math.ceil(filteredUsers.length / PAGE_SIZE))
  const safePage = Math.min(page, totalPages - 1)
  const pagedUsers = useMemo(() => {
    const start = safePage * PAGE_SIZE
    return filteredUsers.slice(start, start + PAGE_SIZE)
  }, [filteredUsers, safePage])

  const createMutation = useMutation({
    mutationFn: (payload: typeof newUser) => api.post("/users", payload),
    onSuccess: () => {
      toast.success("Usuário criado")
      setCreateOpen(false)
      setNewUser({ username: "", password: "", role: "user", name: "" })
      queryClient.invalidateQueries({ queryKey: ["users"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/users/${id}`),
    onSuccess: () => {
      toast.success("Usuário excluído")
      queryClient.invalidateQueries({ queryKey: ["users"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: number; role: string }) =>
      api.patch(`/users/${id}`, { role }),
    onSuccess: () => {
      toast.success("Role atualizado")
      queryClient.invalidateQueries({ queryKey: ["users"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const handleCreate = () => createMutation.mutate(newUser)

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-8 w-28" />
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Nome</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="w-[100px]">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-8 w-28" /></TableCell>
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
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3 flex-1 min-w-48">
          <div className="relative flex-1 max-w-64">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Buscar usuário..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(0) }}
              className="pl-8"
            />
          </div>
          <span className="text-sm text-muted-foreground">
            {filteredUsers.length} usuário(s)
          </span>
        </div>
        {perms.canManageUsers && (
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4" />
                Criar Usuário
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Criar Usuário</DialogTitle>
                <DialogDescription>
                  Novo usuário com role admin ou user. Role owner é reservado (único, via onboarding).
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="username">Username</Label>
                  <Input
                    id="username"
                    value={newUser.username}
                    onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="name">Nome (opcional)</Label>
                  <Input
                    id="name"
                    value={newUser.name}
                    onChange={(e) => setNewUser({ ...newUser, name: e.target.value })}
                    placeholder="Ex: João Silva"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">Senha</Label>
                  <Input
                    id="password"
                    type="password"
                    value={newUser.password}
                    onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Role</Label>
                  <Select value={newUser.role} onValueChange={(v) => setNewUser({ ...newUser, role: v })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="admin">admin</SelectItem>
                      <SelectItem value="user">user</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setCreateOpen(false)}>Cancelar</Button>
                <Button onClick={handleCreate} disabled={!newUser.username || !newUser.password || createMutation.isPending}>
                  {createMutation.isPending ? "Criando..." : "Criar"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
      </div>

      {filteredUsers.length === 0 ? (
        <div className="rounded-md border">
          <EmptyState
            icon={UsersIcon}
            message={search ? "Nenhum usuário encontrado" : "Nenhum usuário cadastrado"}
            action={
              !search && perms.canManageUsers ? (
                <Button size="sm" onClick={() => setCreateOpen(true)}>
                  <Plus className="h-4 w-4" />
                  Criar Usuário
                </Button>
              ) : undefined
            }
          />
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Nome</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="w-[100px]">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pagedUsers.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium">{u.username}</TableCell>
                  <TableCell className="text-muted-foreground">{u.name || "—"}</TableCell>
                  <TableCell>
                    {u.role === "owner" ? (
                      <span className="flex items-center gap-1.5 text-primary font-medium">
                        <ShieldCheck className="h-4 w-4" />
                        owner
                      </span>
                    ) : (
                      <Select
                        value={u.role}
                        onValueChange={(v) => roleMutation.mutate({ id: u.id, role: v })}
                        disabled={!perms.canManageUsers}
                      >
                        <SelectTrigger className="w-32">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="admin">admin</SelectItem>
                          <SelectItem value="user">user</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </TableCell>
                  <TableCell>
                    {u.role !== "owner" && perms.canDeleteUsers && (
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Excluir usuário "{u.username}"?</AlertDialogTitle>
                            <AlertDialogDescription>
                              Esta ação é irreversível. O usuário perderá acesso imediatamente.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancelar</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => deleteMutation.mutate(u.id)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              Excluir
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
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-2 border-t">
              <span className="text-xs text-muted-foreground">
                Página {safePage + 1} de {totalPages}
              </span>
              <div className="flex gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={safePage === 0}
                  aria-label="Página anterior"
                >
                  <ChevronLeft className="h-3.5 w-3.5" />
                  Anterior
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                  disabled={safePage >= totalPages - 1}
                  aria-label="Próxima página"
                >
                  Próxima
                  <ChevronRight className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
