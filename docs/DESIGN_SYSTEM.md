# RESMA — Design System

> Documento de referência para agentes e desenvolvedores. Todas as decisões visuais do RESMA estão consolidadas aqui.
> Alinhado com o AI Engineering Framework (`@.ai`) — Design Mode: **GREENFIELD** (dark-only).

---

## 1. Stack Visual

| Item | Valor |
|------|-------|
| Framework CSS | Tailwind CSS v4 (CSS-first, `@theme` directive) |
| Biblioteca UI | shadcn/ui (componentes customizados) |
| Framework JS | React 19 + TypeScript + Vite |
| Roteamento | React Router v7 |
| Ícones | Lucide React |
| Charts | Recharts |
| Tema | Dark-only (sem light mode) |
| Fonte | System UI stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`) |
| Font size base | 16px |
| Font features | `cv02, cv03, cv04, cv11` |

---

## 2. Tokens de Cor

> Todas as cores em **OKLCH** (oklch(L C H / alpha)). Definidas no `@theme` block do `index.css` para que o Tailwind v4 gere utility classes automaticamente.

### 2.1 Superfícies

| Token | Valor OKLCH | Uso |
|-------|-------------|-----|
| `--color-background` | `oklch(0.16 0.005 260)` | Fundo da página |
| `--color-foreground` | `oklch(0.96 0.003 260)` | Texto principal |
| `--color-card` | `oklch(0.19 0.006 260)` | Background de cards |
| `--color-card-foreground` | `oklch(0.96 0.003 260)` | Texto em cards |
| `--color-popover` | `oklch(0.19 0.006 260)` | Background de popovers/dropdowns |
| `--color-popover-foreground` | `oklch(0.96 0.003 260)` | Texto em popovers |
| `--color-sidebar` | `oklch(0.17 0.005 260)` | Background da sidebar |
| `--color-sidebar-foreground` | `oklch(0.96 0.003 260)` | Texto na sidebar |

### 2.2 Cores Semânticas

| Token | Valor OKLCH | Uso |
|-------|-------------|-----|
| `--color-primary` | `oklch(0.62 0.19 255)` | Azul — CTA, links, foco |
| `--color-primary-foreground` | `oklch(0.98 0 0)` | Texto sobre primary |
| `--color-secondary` | `oklch(0.24 0.006 260)` | Superfície secundária |
| `--color-secondary-foreground` | `oklch(0.96 0.003 260)` | Texto sobre secondary |
| `--color-muted` | `oklch(0.22 0.005 260)` | Background de itens muted |
| `--color-muted-foreground` | `oklch(0.62 0.01 260)` | Texto secundário/metadata |
| `--color-accent` | `oklch(0.26 0.008 260)` | Hover de itens |
| `--color-accent-foreground` | `oklch(0.96 0.003 260)` | Texto sobre accent |
| `--color-destructive` | `oklch(0.58 0.2 25)` | Vermelho — erro, perigo |
| `--color-destructive-foreground` | `oklch(0.98 0 0)` | Texto sobre destructive |
| `--color-success` | `oklch(0.6 0.15 160)` | Verde — sucesso, coleta ativa |
| `--color-success-foreground` | `oklch(0.98 0 0)` | Texto sobre success |
| `--color-warning` | `oklch(0.7 0.16 70)` | Amarelo — alerta, atenção |
| `--color-warning-foreground` | `oklch(0.2 0 0)` | Texto sobre warning (escuro) |

### 2.3 Bordas e Inputs

| Token | Valor OKLCH | Uso |
|-------|-------------|-----|
| `--color-border` | `oklch(0.26 0.006 260 / 0.6)` | Bordas com 60% opacidade — suave |
| `--color-input` | `oklch(0.22 0.005 260)` | Background de inputs |
| `--color-ring` | `oklch(0.62 0.19 255)` | Cor de foco (focus ring) |
| `--color-sidebar-border` | `oklch(0.24 0.006 260 / 0.6)` | Bordas da sidebar com 60% opacidade |

### 2.4 Charts (5 cores)

| Token | Valor OKLCH | Cor aproximada | Uso |
|-------|-------------|----------------|-----|
| `--color-chart-1` | `oklch(0.62 0.19 255)` | Azul | CPU, série primária |
| `--color-chart-2` | `oklch(0.6 0.15 160)` | Verde | Memória, containers saudáveis |
| `--color-chart-3` | `oklch(0.7 0.16 70)` | Amarelo | Alertas, atenção |
| `--color-chart-4` | `oklch(0.58 0.2 25)` | Vermelho | Erros, crítico |
| `--color-chart-5` | `oklch(0.65 0.18 300)` | Roxo | Templates, série especial |

### 2.5 Arquitetura de Cores (Tailwind v4)

```
@theme { --color-* }  →  gera utility classes (text-primary, bg-success, etc.)
:root { --* }         →  alias via var() para compatibilidade com shadcn/ui
```

**Regra crítica**: nunca definir cores diretamente em `:root` sem o bloco `@theme`. O Tailwind v4 só gera utility classes a partir de `--color-*` dentro de `@theme`.

---

## 3. Tipografia

| Nível | Tamanho | Uso |
|-------|---------|-----|
| Page title | `text-2xl font-bold tracking-tight` | Títulos de página |
| Section header | `text-lg font-semibold` | Cabeçalhos de seção |
| Card title | `text-sm font-medium` | Títulos de card |
| Body | `text-sm` | Texto padrão |
| Metadata | `text-xs text-muted-foreground` | Labels, timestamps, info secundária |
| Badge | `text-xs font-semibold` | Badges e tags |
| Micro | `text-[10px] text-muted-foreground` | Subtítulos compactos |

---

## 4. Espaçamento e Layout

| Token | Valor | Uso |
|-------|-------|-----|
| `--radius` | `0.625rem` (10px) | Raio base |
| Card radius | `rounded-xl` (12px) | Cards |
| Button radius | `rounded-md` (8px) | Botões |
| Badge radius | `rounded-md` (8px) | Badges |
| Sidebar width | `w-60` (expandida) / `w-16` (colapsada) | Sidebar |
| Page padding | `p-6` | Padding de página |
| Section gap | `space-y-6` | Entre seções de página |
| Card gap | `gap-4` | Entre cards em grid |

---

## 5. Sidebar — Cores por Rota

Cada item de navegação tem cor própria para identificar a rota ativa:

| Rota | Label | Ícone | activeColor | activeBg | barColor |
|------|-------|-------|-------------|----------|----------|
| `/` | Dashboard | LayoutDashboard | `text-primary` (azul) | `bg-primary/10` | `bg-primary` |
| `/recommendations` | Recomendações | Lightbulb | `text-warning` (amarelo) | `bg-warning/10` | `bg-warning` |
| `/templates` | Templates | FileCode | `text-chart-5` (roxo) | `bg-chart-5/10` | `bg-chart-5` |
| `/services` | Serviços | Boxes | `text-chart-2` (verde) | `bg-chart-2/10` | `bg-chart-2` |

**Barra ativa**: `absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full` com `barColor`.
**Transição**: `transition-all` em todos os NavLinks.

---

## 6. Componentes — Padrões de Estilo

### 6.1 Card

```tsx
// Base — sempre com transition suave
<div className="rounded-xl border bg-card text-card-foreground shadow transition-all duration-200" />

// Clicável — hover com borda primária, sombra e lift
className="cursor-pointer hover:border-primary/50 hover:shadow-lg hover:-translate-y-0.5 transition-all"

// Não-clicável — hover sutil
className="hover:bg-accent/30 transition-colors"

// Stat card (ServiceDetail, ContainerDetail)
className="hover:border-primary/40 hover:shadow-md transition-all"
```

### 6.2 Button

```tsx
// Base — transition suave + active scale
"transition-all duration-150 active:scale-[0.97]"

// Cursor pointer global
button:not(:disabled) { cursor: pointer; }
```

### 6.3 Table Row

```tsx
// Base — hover com bg accent suave
"border-b transition-colors duration-150 hover:bg-accent/40 data-[state=selected]:bg-muted"

// Linha clicável (Services, ServiceDetail containers)
className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
```

### 6.4 Tabs

```tsx
// Tab ativa — borda inferior primária + fundo
"bg-background text-foreground shadow border-b-2 border-primary rounded-b-none"

// Tab inativa
"hover:text-foreground"
```

### 6.5 Badge Semântico (container count)

```tsx
// Regra de cores por quantidade de containers
count > 5  → "border-destructive/40 text-destructive"  // vermelho
count > 2  → "border-warning/40 text-warning"           // amarelo
count <= 2 → "border-chart-2/40 text-chart-2"           // verde
```

### 6.6 Badge "Coletando" (pulse dot)

```tsx
<Badge variant="outline" className="gap-1.5">
  <span className="relative flex h-2 w-2">
    <span className="absolute inline-flex h-full w-full rounded-full bg-success animate-pulse-dot" />
  </span>
  Coletando
</Badge>
```

### 6.7 Stat Card — Números Coloridos

Cada stat card no Dashboard usa uma cor de destaque para o número:

| Stat | Cor | Token |
|------|-----|-------|
| Serviços | Verde | `text-chart-2` |
| Containers | Azul | `text-primary` |
| Recomendações | Amarelo | `text-warning` |
| CPU P95 | Roxo | `text-chart-5` |

---

## 7. Animações

### 7.1 Pulse Dot (coleta ativa)

```css
@keyframes pulse-dot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.7); }
}

.animate-pulse-dot {
  animation: pulse-dot 2s ease-in-out infinite;
}
```

### 7.2 Transições Globais

| Elemento | Propriedades | Duração |
|----------|-------------|---------|
| Card | `border-color, box-shadow, transform` | `0.2s ease` |
| Table row | `background-color` | `0.15s ease` |
| Button | `all` | `0.15s` |
| NavLink | `all` | `transition-all` |

---

## 8. Scrollbar

```css
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: oklch(0.62 0.19 255 / 0.3); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--primary); }
```

---

## 9. Bordas — Regra de Transparência

**Princípio**: bordas no tema dark devem ser suaves, não brancas.

```css
@layer base {
  .border, .border-b, .border-t, .border-r, .border-l {
    border-color: var(--color-border);
  }
}
```

- `--color-border` usa `oklch(0.26 0.006 260 / 0.6)` — escuro com 60% opacidade
- `--color-sidebar-border` usa `oklch(0.24 0.006 260 / 0.6)` — mesmo princípio
- Badges com cores específicas (ex: `border-chart-2/40`) **não** são sobrescritos pelo `@layer base` pois suas classes têm maior especificidade

---

## 10. Alinhamento de Tabelas

- Colunas de **CPU** e **Memória P95**: alinhadas à direita (`text-right` no `<th>` e `<td>`)
- Valores com `whitespace-nowrap` para evitar quebra de linha
- Headers também com `text-right` para alinhar com os valores

---

## 11. PageHeader

Padrão de cabeçalho de página usado em todas as páginas:

```tsx
<PageHeader title="..." description="...">
  <Badge variant="outline" className="gap-1.5">
    <span className="relative flex h-2 w-2">
      <span className="absolute inline-flex h-full w-full rounded-full bg-success animate-pulse-dot" />
    </span>
    Coletando
  </Badge>
  <Badge variant="outline">{count} itens</Badge>
</PageHeader>
```

---

## 12. Breadcrumbs

Construídos dinamicamente a partir do path:

| Path | Breadcrumbs |
|------|-------------|
| `/` | RESMA |
| `/services` | RESMA > Serviços |
| `/services/{name}` | RESMA > Serviços > {name} |
| `/services/{name}/containers/{id}` | RESMA > Serviços > {name} > Container {id.substring(0,12)} |
| `/recommendations` | RESMA > Recomendações |
| `/templates` | RESMA > Templates |

---

## 13. Estados Vazios

- Ícone centralizado com `text-muted-foreground`
- Mensagem em `text-sm text-muted-foreground`
- Padding generoso (`py-12`)
- Sem borda ou card — apenas conteúdo centralizado

---

## 14. Grid Responsivo

| Breakpoint | Grid de stat cards |
|------------|-------------------|
| Mobile | 1 coluna |
| `sm` | 2 colunas |
| `lg` | 4 colunas |

```tsx
<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
```

---

## 15. Arquivos de Referência

| Arquivo | Responsabilidade |
|---------|-----------------|
| `frontend/src/index.css` | Tokens, animações, estilos globais |
| `frontend/src/components/ui/card.tsx` | Componente Card base |
| `frontend/src/components/ui/button.tsx` | Componente Button base |
| `frontend/src/components/ui/table.tsx` | Componente Table base |
| `frontend/src/components/ui/tabs.tsx` | Componente Tabs com borda ativa |
| `frontend/src/components/Layout.tsx` | Sidebar, breadcrumbs, navegação |
| `frontend/src/pages/Dashboard.tsx` | Dashboard com stat cards coloridos |
| `frontend/src/pages/Services.tsx` | Tabela com badges semânticos |
| `frontend/src/pages/ServiceDetail.tsx` | Detalhe com tabs e stat cards |
| `frontend/src/pages/ContainerDetail.tsx` | Detalhe do container |

---

## 16. Princípios de Design

1. **Dark-only**: o app foi desenhado para tema escuro. Não há light mode.
2. **Bordas suaves**: sempre com transparência (60%) para evitar contraste gritante.
3. **Cor com semântica**: cada cor tem significado (azul=primário, verde=saudável, amarelo=alerta, vermelho=crítico, roxo=especial).
4. **Feedback imediato**: todo elemento interativo tem hover e/ou transition.
5. **Cursor pointer**: todo botão clicável indica interatividade.
6. **Densidade moderada**: SaaS B2B, informações densas mas legíveis.
7. **Consistência**: usar sempre os tokens, nunca hardcodear cores.
8. **Acessibilidade**: contraste de texto ≥ 4.5:1 (WCAG AA).

---

## 17. Regras Primordiais — shadcn/ui

> **INEGOCIÁVEIS.** Qualquer implementação de UI deve seguir estas regras sem exceção.

1. **Sempre usar componentes shadcn/ui** — nunca criar componentes UI do zero quando existe um equivalente no shadcn.
2. **Instalar via CLI** — `npx shadcn@latest add <component>`. Nunca copiar código manualmente.
3. **Nunca alterar um componente original shadcn** — os arquivos em `src/components/ui/*` são intocáveis.
4. **Customizar criando novos componentes** — composição sobre modificação. Criar um novo arquivo (ex: `src/components/combobox.tsx`) que importa e compõe os originais.
5. **Combobox = Popover + Command** — para selects pesquisáveis, sempre usar a composição oficial `Popover` + `Command` do shadcn, nunca `<select>` nativo.
6. **Consultar documentação atualizada** — usar context7 ou web search para verificar padrões atuais antes de implementar.
7. **Proibido `<select>` nativo** — em qualquer lugar da aplicação. Sempre usar Combobox ou Select do shadcn.
