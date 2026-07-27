import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// Quando rodando em Docker, o target do proxy deve ser o nome do serviço (go-dev:8080).
// Quando rodando no host, deve ser localhost:8080.
const apiTarget = process.env.VITE_PROXY_TARGET || "http://localhost:8080"

// VITE_PROXY_TARGET é setada apenas no docker-compose.yml — serve como sinal
// de que estamos rodando dentro do container com bind mount do Windows.
const inDocker = !!process.env.VITE_PROXY_TARGET

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: true, // necessário para acessível de fora do container
    // WSL2 + Docker: bind mount do Windows não propaga eventos inotify para o
    // container Linux (microsoft/WSL#4739), então o chokidar do Vite nunca
    // recebe notificação de mudança → HMR não dispara. Polling contorna isso
    // fazendo stat periódico dos arquivos. Só ativado em Docker para preservar
    // o HMR nativo (inotify) e menor CPU quando roda `pnpm dev` no host.
    //
    // ignored: o Vite ignora node_modules por padrão, mas NÃO ignora
    // .pnpm-store (cache do pnpm no bind mount com ~17000 arquivos) nem
    // dist (build output). Sem ignored, o polling faz stat de 17000+ arquivos
    // a cada interval, causando ~40% CPU constante no frontend-dev.
    //
    // interval=1000 (1s) é o equilíbrio entre HMR responsivo e CPU baixa.
    // Doc: https://vite.dev/config/server-options.html#server-watch
    watch: inDocker ? {
      usePolling: true,
      interval: 1000,
      ignored: ["**/.pnpm-store/**", "**/dist/**", "**/.git/**"],
    } : undefined,
    proxy: {
      // SSE proxy — deve vir antes de /api para matching correto.
      // x-accel-buffering: no desabilita buffering do proxy para streaming.
      // timeout/proxyTimeout: 0 desabilita timeout do proxy para SSE (conexões
      // long-lived que o http-proxy fecharia prematuramente após ~30s).
      "/api/sse": {
        target: apiTarget,
        changeOrigin: true,
        timeout: 0,
        proxyTimeout: 0,
        // Workaround para bugs conhecidos do Vite proxy com SSE:
        //   - vitejs/vite#12157: SSE close event não é forwardado
        //   - vitejs/vite#13522: proxy não fecha conexão quando cliente desconecta
        //   - vitejs/vite#20712: abort do servidor não é forwardado ao cliente
        // O http-proxy (usado pelo Vite) não encerra o res do cliente quando o
        // proxyRes fecha. Sem isso, o EventSource fica em limbo sem receber dados.
        configure: (proxy) => {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.setHeader("x-accel-buffering", "no")
          })
          // Forward close/abort do backend → cliente (vitejs/vite#20712)
          proxy.on("proxyRes", (proxyRes, req, res) => {
            proxyRes.on("close", () => {
              if (!res.writableEnded) {
                res.destroy()
              }
            })
            proxyRes.on("error", (err) => {
              if (!res.writableEnded) {
                res.destroy(err)
              }
            })
          })
        },
      },
      "/api": apiTarget,
    },
  },
})
