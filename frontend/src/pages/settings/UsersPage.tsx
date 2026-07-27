import { useState, useEffect } from "react"
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
import { Plus, Trash2, ShieldCheck } from "lucide-react"
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
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: "", password: "", role: "user", name: "" })

  const loadUsers = async () => {
    try {
      const data = await api.get<{ users: User[] }>("/users")
      setUsers(data.users || [])
    } catch (e) {
      toast.error("Erro ao carregar usuários: " + (e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadUsers()
  }, [])

  const handleCreate = async () => {
    try {
      await api.post("/users", newUser)
      toast.success("Usuário criado")
      setCreateOpen(false)
      setNewUser({ username: "", password: "", role: "user", name: "" })
      loadUsers()
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    }
  }

  const handleDelete = async (id: number, username: string) => {
    if (!confirm(`Excluir usuário "${username}"?`)) return
    try {
      await api.delete(`/users/${id}`)
      toast.success("Usuário excluído")
      loadUsers()
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    }
  }

  const handleChangeRole = async (id: number, role: string) => {
    try {
      await api.patch(`/users/${id}`, { role })
      toast.success("Role atualizado")
      loadUsers()
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    }
  }

  if (loading) return <div className="text-muted-foreground">Carregando...</div>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{users.length} usuário(s)</p>
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
                <Button onClick={handleCreate} disabled={!newUser.username || !newUser.password}>
                  Criar
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
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
            {users.map((u) => (
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
                      onValueChange={(v) => handleChangeRole(u.id, v)}
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
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDelete(u.id, u.username)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
