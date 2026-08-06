# RESMA CLI — Wireframe TUI (`cmd/wire-tui`)

> **Status:** Wireframe HD funcional com dados mockados
> **Comando:** `go run ./cmd/wire-tui`
> **Build:** `go build -o wire-tui.exe ./cmd/wire-tui/`

## Objetivo

Wireframe de alta fidelidade do dashboard TUI do `resma monitor`.
Demonstra o layout de produção (two-column, 6 tabs, drill-down, filter/command modes)
com dados mockados, sem precisar de API real.

## Estrutura de arquivos (cada arquivo pequeno para não crashar IDE)

```
cmd/wire-tui/
├── main.go                  # entrypoint (21 linhas)
├── model.go                 # struct + Init/Update/View (104 linhas)
├── keys.go                  # handleKey + command/filter execution (194 linhas)
├── styles.go                # estilos lipgloss (95 linhas)
├── mockdata.go              # dados mockados (115 linhas)
├── sparkline.go             # helper sparkline + pad/truncate (59 linhas)
├── layout.go                # renderDashboard + renderListView + helpers (197 linhas)
├── header.go                # renderHeader (43 linhas)
├── tabbar.go                # renderTabBar (15 linhas)
├── footer.go                # renderFooter contextual (31 linhas)
├── breadcrumb.go            # renderBreadcrumb (12 linhas)
├── services_tab.go          # tab [1] Services (64 linhas)
├── nodes_tab.go             # tab [2] Nodes (64 linhas)
├── agents_tab.go            # tab [3] Agents (44 linhas)
├── tasks_tab.go             # tab [4] Tasks (46 linhas)
├── alerts_tab.go            # tab [5] Alerts (47 linhas)
├── recommendations_tab.go   # tab [6] Recommendations (64 linhas)
├── detail_view.go           # 6 detail views para drill-down (132 linhas)
└── input_help.go            # command/filter input + help overlay (90 linhas)
```

## Features implementadas no wireframe

- [x] Layout two-column (side panel 33% + main panel 67%)
- [x] Bordas com indicação de foco (ativa roxa, inativa cinza)
- [x] Header com título + relógio + status SSE
- [x] Tab bar com 6 tabs e highlight na ativa
- [x] Navegação j/k, g/G, Tab/Shift+Tab, 1-6
- [x] Drill-down com Enter (6 detail views) + Esc para voltar
- [x] Command mode `:` com execução (services, nodes, agents, etc.)
- [x] Filter mode `/` com input
- [x] Help overlay `?` com keybindings contextuais
- [x] Footer contextual (muda por view mode e tab)
- [x] Breadcrumb de navegação
- [x] Sparklines Unicode nas tabelas
- [x] Cores semânticas (success/warning/error/muted)
- [x] Minimum terminal size 80x24 com mensagem de erro
- [x] Alt-screen com restauração ao sair

## Keybindings

| Tecla | Ação |
|-------|------|
| `q` / `Ctrl+c` | Sair (ou fecha overlay) |
| `1`-`6` | Trocar de tab |
| `Tab` / `Shift+Tab` | Trocar painel focado (side ↔ main) |
| `j` / `↓` | Próximo item |
| `k` / `↑` | Item anterior |
| `g` | Primeiro item |
| `G` | Último item |
| `Enter` | Drill-down (detail view) |
| `Esc` | Voltar (detail → list, cancel filter/command) |
| `/` | Filter mode |
| `:` | Command mode |
| `?` | Help overlay |
| `r` | Refresh (mock) |

## Dados mockados

- 8 serviços (7 running, 1 stopped) com sparklines de CPU
- 5 nodes (4 ready, 1 down) com CPU/MEM/DISK
- 5 agents (4 active, 1 offline)
- 12 tasks (10 running, 2 failed)
- 6 alerts (2 critical, 3 warning, 1 info)
- 6 recommendations (2 high, 2 medium, 2 low risk)

## Próximos passos (implementação real)

O wireframe valida o layout e a UX. A implementação real em `internal/tui/` deve:
1. Substituir dados mockados por SSE events reais
2. Usar `bubbles/table` com `AutoTable` (auto-sizing de colunas)
3. Usar `bubbles/list` para o side panel
4. Usar `bubbles/viewport` para detail views
5. Usar `bubbles/textinput` para filter/command modes
6. Implementar skins YAML (estilo k9s)
7. Implementar hotkeys customizáveis
8. Adicionar mouse support opcional
9. Adicionar confirmation modals para ações destrutivas
10. Adicionar toast notifications
