import { useState, type FormEvent } from "react"
import { useAuth } from "@/contexts/AuthContext"
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Activity, ShieldCheck, Server, Cpu, MemoryStick } from "lucide-react"

export default function Onboarding() {
  const { onboarding } = useAuth()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError("")

    if (password !== confirmPassword) {
      setError("As senhas não coincidem")
      return
    }
    if (password.length < 8) {
      setError("A senha deve ter no mínimo 8 caracteres")
      return
    }

    setLoading(true)
    try {
      await onboarding(username, password)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Onboarding failed")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen">
      <div className="relative hidden flex-1 flex-col justify-between bg-gradient-to-br from-primary/20 via-background to-background p-12 lg:flex">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/20">
            <Activity className="h-5 w-5 text-primary" />
          </div>
          <div>
            <span className="text-lg font-bold tracking-tight">RESMA</span>
            <p className="text-xs text-muted-foreground">Otus7 Infrastructure</p>
          </div>
        </div>

        <div className="space-y-6">
          <h2 className="text-3xl font-bold leading-tight">
            Resource Manager<br />para Docker Swarm
          </h2>
          <p className="text-muted-foreground max-w-md">
            Monitoramento, recomendações data-driven e templates de recursos em um único lugar.
          </p>
          <div className="grid grid-cols-3 gap-4 pt-4">
            <div className="space-y-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <Server className="h-5 w-5 text-primary" />
              </div>
              <p className="text-xs text-muted-foreground">Monitoramento<br/>de containers</p>
            </div>
            <div className="space-y-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-chart-2/10">
                <Cpu className="h-5 w-5 text-chart-2" />
              </div>
              <p className="text-xs text-muted-foreground">Recomendações<br/>de CPU/Mem</p>
            </div>
            <div className="space-y-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-chart-3/10">
                <MemoryStick className="h-5 w-5 text-chart-3" />
              </div>
              <p className="text-xs text-muted-foreground">Templates<br/>predefinidos</p>
            </div>
          </div>
        </div>

        <p className="text-xs text-muted-foreground">© 2025 Otus7 — All rights reserved</p>
      </div>

      <div className="flex flex-1 items-center justify-center bg-background px-4">
        <Card className="w-full max-w-md">
          <CardHeader className="space-y-3 text-center">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-primary/15">
              <Activity className="h-6 w-6 text-primary" />
            </div>
            <div>
              <CardTitle className="text-2xl">Bem-vindo ao RESMA</CardTitle>
              <CardDescription className="mt-1 flex items-center justify-center gap-1.5">
                <ShieldCheck className="h-4 w-4" />
                Crie sua conta de administrador
              </CardDescription>
            </div>
          </CardHeader>
          <form onSubmit={handleSubmit}>
            <CardContent className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="space-y-2">
                <Label htmlFor="username">Usuário</Label>
                <Input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="admin"
                  required
                  minLength={3}
                  autoFocus
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">Senha</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Min. 8 caracteres, 1 letra e 1 número"
                  required
                  minLength={8}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirmPassword">Confirmar Senha</Label>
                <Input
                  id="confirmPassword"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="••••••••"
                  required
                  minLength={8}
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "Criando conta..." : "Criar conta"}
              </Button>
            </CardFooter>
          </form>
        </Card>
      </div>
    </div>
  )
}
