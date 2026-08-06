# RESMA CLI — Estudo: Padrões UI/UX do k9s

> **Fonte:** Código fonte do k9s (derailed/k9s) — internal/ui/, internal/view/

## 1. Arquitetura de Layout do k9s

```
┌─────────────────────────────────────────────────────────────────────┐
│  [ClusterInfo]  [Menu/KeyHints]                    [Logo]  [Status]  │ ← Header (7 linhas)
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Content — PageStack]                                              │ ← Main (flex)
│  (Table/List/Detail/Logs/etc)                                       │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  [Crumbs — breadcrumbs]                                             │ ← 1 linha
├─────────────────────────────────────────────────────────────────────┤
│  [Flash — toast messages]                                           │ ← 1 linha
└─────────────────────────────────────────────────────────────────────┘
```

### Componentes do header (7 linhas)

1. **ClusterInfo** (esquerda, 50 chars) — nome do cluster, contexto, versão k8s, CPU/MEM
2. **Menu** (centro, flex) — grid de keyhints em colunas (max 6 rows)
3. **Logo** (direita, 26 chars) — ASCII art do logo + status bar colorida

### Menu (keyhints grid)

O k9s não tem um footer simples — tem um **menu grid** no header que mostra
todos os keybindings ativos em colunas, ordenados. Formato:

```
 <0> Services   <a> Apply       <d> Delete
 <1> Nodes      <e> Edit        <l> Logs
 <2> Agents     <s> Shell       <y> YAML
 <3> Tasks      <r> Refresh     <?> Help
 <4> Alerts     <q> Quit        <:> Command
 <5> Recs       </> Filter      <G> Bottom
```

- Máximo 6 rows
- Colunas organizadas por tipo (navegação, ações, global)
- Números (`<0>`, `<1>`) têm cor diferente de letras (`<a>`, `<d>`)
- Cada hint: `<key> description` com key em bold colorido

### Crumbs (breadcrumbs)

Formato: `[fg:bg:b] <name> [-:bg:-]` — cada crumb é um "chip" com background colorido.
O último crumb tem cor diferente (active). Exemplo:

```
 <services> <api> <containers>
```

### Flash (toast)

Centro-alinhado, com emoji + mensagem:
- Happy 😎 — success
- Doh 😗 — warning
- Red 😡 — error

### Prompt (command/filter)

- Borda colorida (roxa para command `>`, cinza para filter `/`)
- Ícone: 🐶 para command, 🐩 para filter
- Sugestões em tempo real (autocomplete) com cor diferente
- Tab aceita sugestão
- Up/Down navega histórico de sugestões

## 2. Padrões-chave do k9s

### 2.1 PageStack (navegação)

- Stack de páginas — cada view é uma página
- `Enter` empurra nova página (drill-down)
- `Esc`/`[` desempurra (volta)
- `]` vai para frente (histórico)
- `-` vai para última view

### 2.2 Table

- `tview.Table` com `SetFixed(1, 0)` — header fixo
- `SetSelectable(true, false)` — só linhas selecionáveis
- `SetSelectionChangedFunc` — callback quando cursor muda
- Header com bold + cor de destaque
- Linha selecionada com background colorido
- Sort por coluna (click no header ou `s`/`Shift+S`)
- Wide mode (`W`) — mostra colunas extras
- Filter (`/`) com regex
- Column selection (`Shift+Left/Right`) — destaca coluna para sort

### 2.3 KeyHints dinâmicas

Cada view registra suas próprias keyhints. Quando a view muda (drill-down),
o menu atualiza automaticamente. As hints são `model.MenuHints` ordenadas.

### 2.4 Status Indicator

Linha de 1 char no topo mostrando status de conexão:
- Verde: conectado
- Vermelho: desconectado
- Pisca quando reconectando

### 2.5 Splash screen

Mostra splash por 1 segundo no startup, depois transiciona para main.

## 3. O que o nosso wire-tui precisa melhorar

### Problemas atuais vs k9s

| Aspecto | wire-tui atual | k9s | Ação |
|---------|----------------|-----|------|
| Header | 1 linha simples | 7 linhas com ClusterInfo + Menu grid + Logo | Adicionar menu grid de keyhints |
| Footer | 1 linha com texto | Não tem footer (keyhints no header) | Mover keyhints para header |
| Menu/KeyHints | Não tem | Grid 6xN com `<key> desc` | Implementar menu grid |
| Crumbs | Texto simples | Chips coloridos com `<name>` | Estilizar crumbs |
| Flash/Toast | Não tem | Emoji + mensagem centralizada | Implementar flash |
| Prompt | Input simples | Borda + ícone + sugestões | Melhorar prompt |
| Table | fmt.Sprintf | tview.Table com header fixo, sort, selection | Usar bubbles/table |
| Splash | Não tem | Tela inicial por 1s | Adicionar splash |
| Logo | Não tem | ASCII art + status bar | Adicionar logo ASCII |
| Status | Texto no header | Indicator dedicado | Separar status |

### Plano de melhoria

1. **Header rico** — 3 seções: logo+status (esquerda), menu grid (centro), clock (direita)
2. **Menu grid** — keyhints em grid 6xN com `<key> desc` formatado
3. **Crumbs estilizados** — chips coloridos
4. **Flash/toast** — emoji + mensagem, auto-dismiss
5. **Prompt melhorado** — borda + sugestões + autocomplete
6. **Splash screen** — logo ASCII por 1s no startup
7. **Tabelas com bubbles/table** — header fixo, selection, sort visual
8. **Status indicator** — linha dedicada com cor semântica
