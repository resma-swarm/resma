import { useState } from "react"
import { useAuth } from "@/contexts/AuthContext"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { PageHeader } from "@/components/page-header"
import { ShieldCheck } from "lucide-react"
import { toast } from "sonner"

export function Profile() {
  const { user, updateUser } = useAuth()
  const [name, setName] = useState(user?.name || "")
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [savingName, setSavingName] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  const handleSaveName = async () => {
    setSavingName(true)
    try {
      await api.put("/auth/profile", { name })
      toast.success("Nome atualizado")
      updateUser({ name })
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    } finally {
      setSavingName(false)
    }
  }

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      toast.error("As senhas não coincidem")
      return
    }
    if (newPassword.length < 6) {
      toast.error("A nova senha deve ter pelo menos 6 caracteres")
      return
    }
    setSavingPassword(true)
    try {
      await api.post("/auth/change-password", {
        current_password: currentPassword,
        new_password: newPassword,
      })
      toast.success("Senha alterada. Faça login novamente.")
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    } finally {
      setSavingPassword(false)
    }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <PageHeader title="Perfil" description="Gerencie suas informações de conta e segurança" />

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Coluna 1 — Informações do perfil e nome */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Perfil</CardTitle>
              <CardDescription>Informações da sua conta</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-3">
                <Label className="w-24 text-muted-foreground">Username:</Label>
                <span className="font-medium">{user?.username}</span>
              </div>
              <div className="flex items-center gap-3">
                <Label className="w-24 text-muted-foreground">Role:</Label>
                {user?.role === "owner" ? (
                  <span className="flex items-center gap-1.5 text-primary font-medium">
                    <ShieldCheck className="h-4 w-4" />
                    owner
                  </span>
                ) : (
                  <span className="font-medium capitalize">{user?.role}</span>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Nome</CardTitle>
              <CardDescription>Seu nome de exibição (opcional)</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Nome</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Ex: João Silva"
                />
              </div>
              <Button
                onClick={handleSaveName}
                disabled={savingName || name === (user?.name || "")}
              >
                {savingName ? "Salvando..." : "Salvar Nome"}
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Coluna 2 — Alterar senha */}
        <div className="flex flex-col">
          <Card className="flex flex-1 flex-col">
            <CardHeader>
              <CardTitle>Alterar Senha</CardTitle>
              <CardDescription>Digite sua senha atual e a nova senha</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-1 flex-col space-y-4">
              <div className="space-y-2">
                <Label htmlFor="current">Senha Atual</Label>
                <Input
                  id="current"
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new">Nova Senha</Label>
                <Input
                  id="new"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm">Confirmar Nova Senha</Label>
                <Input
                  id="confirm"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                />
              </div>
              <Button
                onClick={handleChangePassword}
                disabled={savingPassword || !currentPassword || !newPassword || !confirmPassword}
                className="mt-auto w-fit"
              >
                {savingPassword ? "Salvando..." : "Alterar Senha"}
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
