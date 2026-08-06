import { lazy, Suspense } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { AuthProvider, useAuth } from "@/contexts/AuthContext"
import { Spinner } from "@/components/ui/spinner"
import { Toaster } from "@/components/ui/sonner"
import { Layout } from "@/components/Layout"
import Login from "@/pages/Login"
import Onboarding from "@/pages/Onboarding"
// Core monitoring routes — eager (usadas com frequência, SSE ativo)
import Dashboard from "@/pages/Dashboard"
import Services from "@/pages/Services"
import ServiceDetail from "@/pages/ServiceDetail"
import ContainerDetail from "@/pages/ContainerDetail"
import Nodes from "@/pages/Nodes"
import NodeDetail from "@/pages/NodeDetail"
// Less frequent routes — lazy loaded (code-split para reduzir bundle inicial)
const Templates = lazy(() => import("@/pages/Templates"))
const Schedules = lazy(() => import("@/pages/Schedules"))
const Tasks = lazy(() => import("@/pages/Tasks"))
const Alerts = lazy(() => import("@/pages/Alerts"))
const Studio = lazy(() => import("@/pages/Studio"))
const RollbackWatches = lazy(() => import("@/pages/RollbackWatches").then(m => ({ default: m.RollbackWatches })))
const Settings = lazy(() => import("@/pages/Settings").then(m => ({ default: m.Settings })))
const UsersPage = lazy(() => import("@/pages/settings/UsersPage").then(m => ({ default: m.UsersPage })))
const ApiKeysPage = lazy(() => import("@/pages/settings/ApiKeysPage").then(m => ({ default: m.ApiKeysPage })))
const ParametersPage = lazy(() => import("@/pages/settings/ParametersPage").then(m => ({ default: m.ParametersPage })))
const DataPage = lazy(() => import("@/pages/settings/DataPage").then(m => ({ default: m.DataPage })))
const Profile = lazy(() => import("@/pages/Profile").then(m => ({ default: m.Profile })))

// Suspense fallback — spinner centralizado para rotas lazy
function RouteSpinner() {
  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <Spinner className="h-8 w-8" />
    </div>
  )
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
})

function AppGate() {
  const { initialized, user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner className="h-8 w-8" />
      </div>
    )
  }

  if (!initialized) {
    return <Onboarding />
  }

  if (!user) {
    return <Login />
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          {/* Core routes — eager loaded */}
          <Route path="/" element={<Dashboard />} />
          <Route path="/services" element={<Services />} />
          <Route path="/services/:name" element={<ServiceDetail />} />
          <Route path="/services/:name/containers/:containerId" element={<ContainerDetail />} />
          <Route path="/nodes" element={<Nodes />} />
          <Route path="/nodes/:nodeId" element={<NodeDetail />} />

          {/* Less frequent routes — lazy loaded with Suspense */}
          <Route path="/optimizations" element={<Suspense fallback={<RouteSpinner />}><Studio /></Suspense>} />
          <Route path="/optimizations/rollback-watches" element={<Suspense fallback={<RouteSpinner />}><RollbackWatches /></Suspense>} />
          <Route path="/templates" element={<Suspense fallback={<RouteSpinner />}><Templates /></Suspense>} />
          <Route path="/schedules" element={<Suspense fallback={<RouteSpinner />}><Schedules /></Suspense>} />
          <Route path="/tasks" element={<Suspense fallback={<RouteSpinner />}><Tasks /></Suspense>} />
          <Route path="/alerts" element={<Suspense fallback={<RouteSpinner />}><Alerts /></Suspense>} />

          {/* Fase 8 — Settings area (nested routes, lazy) */}
          <Route path="/settings" element={<Suspense fallback={<RouteSpinner />}><Settings /></Suspense>}>
            <Route index element={<Navigate to="/settings/users" replace />} />
            <Route path="users" element={<Suspense fallback={<RouteSpinner />}><UsersPage /></Suspense>} />
            <Route path="api-keys" element={<Suspense fallback={<RouteSpinner />}><ApiKeysPage /></Suspense>} />
            <Route path="parameters" element={<Suspense fallback={<RouteSpinner />}><ParametersPage /></Suspense>} />
            <Route path="data" element={<Suspense fallback={<RouteSpinner />}><DataPage /></Suspense>} />
          </Route>

          {/* Fase 8 — Profile */}
          <Route path="/profile" element={<Suspense fallback={<RouteSpinner />}><Profile /></Suspense>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <AppGate />
        <Toaster />
      </AuthProvider>
    </QueryClientProvider>
  )
}
