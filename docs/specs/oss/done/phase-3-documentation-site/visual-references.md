# Referências Visuais — Site de Documentação RESMA

> Análise visual consolidada pelo Orchestrator (Knowledge Mission) com visita aos sites em Jul 2026.

## ⚠️ Princípio fundamental — Identidade visual única

> **O site do RESMA deve ter sua própria identidade visual.** As referências abaixo servem como **inspiração de padrões e estrutura**, não como modelos para copiar. Cada decisão de design deve ser tomada pensando no que faz sentido para o RESMA — sua audiência (DevOps, SREs, administradores de Swarm), seus diferenciais (ML, detecção de leaks, OOM tracking) e sua personalidade como projeto.
>
> **Regras:**
> - Não copiar paletas de cores, tipografia ou layouts de outros projetos
> - Criar uma paleta de cores própria e consistente com o tema do RESMA (resource management, monitoring, Swarm)
> - Desenvolver componentes visuais que comuniquem os diferenciais únicos do RESMA
> - Usar as referências para entender o que funciona bem (padrões de UX, estrutura de informação, componentes úteis) e adaptar — não replicar
> - A identidade visual do RESMA deve ser reconhecível e memorável por si só
>
> **O que pegar das referências:** padrões de UX, estrutura de informação, componentes funcionais (callouts, code tabs, API endpoint), hierarquia visual, fluxo de navegação.
>
> **O que NÃO pegar:** cores, fontes, nomes de seções, copy, layouts específicos, identidade de marca.

## Sites visitados e analisados

### 1. Swarmpit — https://swarmpit.io

**Referência principal para landing page do RESMA.**

**Padrões visuais extraídos:**
- **Hero section:** Título com sintaxe de função `swarmpit()` + tagline curta. Badge "v1.10 stable · open source" acima do título. Parágrafo de 1-2 linhas descrevendo o que é. Dois CTAs: `$ install swarmpit` (primário) e `view on github →` (secundário)
- **Stats badges:** Abaixo dos CTAs — license, stars (3.6k+), docker pulls (50M+). Badges minimalistas com label + valor
- **Terminal mockup:** Hero右侧 com terminal simulado mostrando `docker run` + output animado com `→` e `✓`. Cursor blinking `_` no final
- **Screenshots:** Figura mostrando app em desktop, tablet e mobile — demonstra responsividade
- **Features grid:** 8 features em grid 4x2. Cada feature: número sequencial ("01", "02"...), título em heading h3, parágrafo curto. Seção prefixada com `// features` (estilo código)
- **Install section:** Tabs (stable / latest), code block com botão copy, link "read full installation docs →"
- **Contact form:** Simples — name, email, message, botão "send →"
- **Footer:** 3 colunas — projeto (github, issues), contact (email), navigate (features, install, contact)
- **Design language:** Dark theme, monospace accents, sintaxe de código como decoração (`// features`, `// install`, `// contact`), números sequenciais nas features, tabs para versões

**O que aplicar no RESMA:**
- Hero com terminal mockup mostrando `docker stack deploy` + output
- Stats badges (license MIT, stars, docker pulls)
- Features grid numeradas com `// features` prefix
- Install section com tabs stable/edge
- Dark theme como default

### 2. Cetacean — https://cetacean.mazetti.me

**Referência para minimalismo e clareza.**

**Padrões visuais extraídos:**
- **Hero minimalista:** Emoji 🐋 + título "Cetacean" + 1 linha de descrição + 3 CTAs (Try Demo, Get Started, GitHub)
- **Navbar:** Logo com emoji, links (Docs, API, Demo), search com Ctrl+K, version badge, GitHub link, theme toggle
- **Design:** Muito limpo, sem excesso de informações. Foco no CTA "Try Demo" — permite testar antes de instalar
- **Dark mode:** Toggle system/light/dark

**O que aplicar no RESMA:**
- Search com Ctrl+K (Docusaurus já suporta nativamente)
- Version badge na navbar
- CTA "Try Demo" se viável (demo online do dashboard)

### 3. Scribe — https://scribedocs.vercel.app

**Referência para componentes de documentação (alternativa Mintlify).**

**Padrões visuais extraídos:**
- **Sidebar:** Categorias colapsáveis com badges, links externos indicados
- **Callouts:** 5 tipos (info, success, warning, danger, pro tip) — cada um com ícone e cor distintos
- **Code blocks:** Com filename header, botão copy, syntax highlighting. Tabs para múltiplas linguagens (cURL, JS, Python, Go)
- **API Endpoint component:** Badge HTTP method (POST em verde), path em mono, badge "Auth required". Tabela de parâmetros com columns: Parameter, Type, Required, Description, Example
- **Steps:** Numerados com código por step
- **Cards:** Grid de navegação com ícones, descrições, badges (New, Pro)
- **TOC:** Sidebar direita com "On this page" — scroll-tracking com active state
- **Search:** Botão "Search docs" com atalho `/`
- **Version switcher:** Dropdown na navbar
- **Dark mode:** Toggle sem flash, system-aware

**O que aplicar no RESMA:**
- Callouts para docs (info, warning, danger, tip)
- Code tabs para exemplos em múltiplas linguagens (bash, python, curl)
- API Endpoint component para documentação da REST API
- Cards de navegação na homepage das docs
- Search com atalho `/`

### 4. Verdaccio — https://verdaccio.org

**Referência para projeto OSS self-hosted (Docusaurus).**

**Padrões visuais extraídos:**
- **Hero:** Logo + título + tagline + npm install command com copy button. Sem terminal mockup — mais direto
- **Sponsored by:** Logos de sponsors (Docker, JetBrains, Algolia)
- **Used by:** Logos de empresas que usam (nx, pnpm, Angular CLI, Storybook)
- **Feature cards:** Grid de features com ícones
- **Navbar:** Docs, Blog, Community, Metrics + Sponsor Us, GitHub, Bluesky, theme toggle, search
- **Banner:** Link para fundraiser (Ukraine)
- **Footer:** 3 colunas (Docs, Community, More)

**O que aplicar no RESMA:**
- "Used by" section se houver empresas usando
- Command + copy button no hero (em vez de terminal mockup — mais simples)
- Sponsor link (Open Collective)
- Metrics page (downloads, stars)

### 5. Quickwit — https://quickwit.io

**Referência para infra OSS com design polished.**

**Padrões visuais extraídos:**
- **Hero:** Título grande com quebra de linha estilizada ("Search more / Sub-second search & analytics engine on cloud storage with less"). 2 CTAs (Try Quickwit now, Book a demo)
- **Feature blocks:** 4 seções com heading + bullet list, sem cards — mais textual
- **Logos carousel:** Logos de clientes em carousel infinito (2 rows)
- **Architecture diagram:** Imagem mostrando arquitetura do produto
- **Testimonials:** Grid de depoimentos de usuários
- **Open source section:** "Open and Free Community Based Software" com CTAs GitHub + Discord
- **Newsletter:** Input de email no footer
- **Footer:** 3 colunas (Docs, Community, Company) + newsletter

**O que aplicar no RESMA:**
- Architecture diagram (arquitetura do RESMA no Swarm)
- "Open and Free" section com CTAs GitHub + Discord
- Newsletter signup no footer

### 6. Dyte — https://docs.dyte.io

**Referência para docs com Tailwind + multi-SDK + versioning.**

**Padrões visuais extraídos:**
- **Homepage de docs:** Não é apenas docs — é uma landing page dentro do docs
- **Hero:** "Build with Dyte" + parágrafo + 3 cards de features principais (Live Video, Voice, Live Streaming) com links
- **SDK section:** "Build the way you want in the framework you want!" + 2 cards (UI Kit, Core SDK)
- **API Reference section:** Heading + descrição + link "Get started →" + preview image
- **Community section:** Avatares de contributors + links Twitter/LinkedIn
- **Navbar:** Guides, SDKs (dropdown), REST API, Resources (dropdown), Support + Book a demo, Sign Up, theme toggle
- **Design:** TailwindCSS, cards com hover effects, clean e moderno

**O que aplicar no RESMA:**
- Homepage de docs como landing page (não apenas índice)
- Cards de features com links para guias
- Section "API Reference" com preview
- Community section com avatares de contributors

### 7. Docusaurus OpenAPI Docs — https://docusaurus-openapi.tryingpan.dev

**Referência para integração OpenAPI no Docusaurus.**

**Padrões visuais extraídos:**
- **Sidebar organizada:** DOCUMENTATION header, links com categorias colapsáveis
- **Badges:** License, npm version, npm downloads, GitHub stars — no topo da page
- **Color palette selector:** Dropdown "Evergreen" — permite trocar paleta de cores do tema
- **Breadcrumbs:** Navegação breadcrumb no topo de cada page
- **TOC:** Sidebar direita com links ancorados
- **Tags:** Tags no final de cada page (documentation, openapi, getting started)
- **Edit this page:** Link para editar no GitHub
- **Pagination:** "Next →" no final da page
- **Footer:** 3 colunas (Docs, Community, More)

**O que aplicar no RESMA:**
- Color palette customizada (não azul padrão Docusaurus)
- Breadcrumbs
- Tags nas pages
- "Edit this page" link
- Pagination entre pages

## Síntese — Padrão visual recomendado para RESMA

### Landing page (homepage)

```
┌─────────────────────────────────────────────────────┐
│ Navbar: RESMA | Docs | API | GitHub | Theme | Search│
├─────────────────────────────────────────────────────┤
│                                                     │
│  v0.1.0 beta · open source                          │
│                                                     │
│  resma()                                            │
│  Resource Manager for Docker Swarm                  │
│                                                     │
│  ML-powered resource recommendations, memory leak   │
│  detection, and OOM tracking for Docker Swarm.      │
│  2 containers, no external DB, no agent.            │
│                                                     │
│  [$ install resma]  [view on github →]              │
│                                                     │
│  license: MIT  stars: ⭐  docker pulls: 📦          │
│                                                     │
│                    ┌─────────────────────┐          │
│                    │ ~/resma             │          │
│                    │ $ docker stack ...  │          │
│                    │ → deploying...      │          │
│                    │ ✓ ready on :8080    │          │
│                    └─────────────────────┘          │
│                                                     │
├─────────────────────────────────────────────────────┤
│  // features                                        │
│                                                     │
│  01 ML Recommendations    02 Memory Leak Detection │
│  03 OOM Tracking          04 Scheduled Changes      │
│  05 Resource Monitoring   06 Change Log / Audit     │
│  07 REST API              08 Templates YAML         │
│                                                     │
├─────────────────────────────────────────────────────┤
│  // comparison                                      │
│  Tabela: RESMA vs Swarmpit vs Portainer vs Cetacean │
├─────────────────────────────────────────────────────┤
│  // install                                         │
│  [stable] [edge]                                    │
│  $ curl -fsSL .../install.sh | bash                 │
│  read full installation docs →                      │
├─────────────────────────────────────────────────────┤
│  // architecture                                    │
│  [diagrama de arquitetura no Swarm]                 │
├─────────────────────────────────────────────────────┤
│  Open and Free · MIT License                        │
│  [GitHub] [Discord]                                 │
├─────────────────────────────────────────────────────┤
│ Footer: Docs | Community | More | © 2026            │
└─────────────────────────────────────────────────────┘
```

### Docs page (Docusaurus)

```
┌─────────────────────────────────────────────────────┐
│ Navbar: RESMA | Docs | API | GitHub | Theme | Search│
├──────────┬──────────────────────────┬───────────────┤
│ Sidebar  │ Main content             │ On this page  │
│          │                          │               │
│ GETTING  │ Breadcrumb: Home > X     │ - Section 1   │
│ STARTED  │                          │ - Section 2   │
│  Intro   │ # Heading                │ - Section 3   │
│  Install │                          │               │
│  Config  │ > Callout (info)         │               │
│          │                          │               │
│ GUIDES   │ ```bash                  │               │
│  ...     │ $ command                │               │
│          │ ```                      │               │
│ API      │                          │               │
│  ...     │ Tags: docker, swarm      │               │
│          │ Edit this page | Next →  │               │
├──────────┴──────────────────────────┴───────────────┤
│ Footer: Docs | Community | More | © 2026            │
└─────────────────────────────────────────────────────┘
```

### Paleta de cores sugerida

| Elemento | Cor | Hex |
|----------|-----|-----|
| Primary (light) | Azul-petróleo | `#0D9488` (teal-600) ou `#2563EB` (blue-600) |
| Primary (dark) | Teal claro | `#2DD4BF` (teal-400) ou `#60A5FA` (blue-400) |
| Background (dark) | Cinza-escuro | `#0F172A` (slate-900) |
| Background (light) | Branco quente | `#FAFAF9` (stone-50) |
| Accent | Verde | `#10B981` (emerald-500) — para success/healthy |
| Warning | Âmbar | `#F59E0B` (amber-500) |
| Danger | Vermelho | `#EF4444` (red-500) |

### Componentes MDX recomendados

| Componente | Inspiração | Uso |
|------------|-----------|-----|
| Callout (info/warning/danger/tip) | Scribe | Notas e avisos nas docs |
| CodeTabs (bash/python/curl) | Scribe | Exemplos multi-linguagem |
| FeatureCard | Dyte, CCLEE | Cards de features na homepage |
| ComparisonTable | CCLEE | Tabela RESMA vs concorrentes |
| Steps | Scribe | Guias step-by-step |
| ApiEndpoint | Scribe, OpenAPI plugin | Documentação da REST API |
| TerminalMockup | Swarmpit | Hero da landing page |

### Stack técnica confirmada

| Camada | Tecnologia | Justificativa |
|--------|-----------|---------------|
| Framework | Docusaurus 3 | React-based, versioning, i18n, OpenAPI plugin |
| Styling | TailwindCSS 4 | Mesma stack do frontend RESMA |
| Components | shadcn/ui | Mesma stack do frontend RESMA |
| Search | Algolia DocSearch (gratuito para OSS) | Padrão Docusaurus |
| OpenAPI | docusaurus-plugin-openapi-docs | Auto-gerado do Go (swaggo/swag) |
| Deploy | GitHub Pages | Gratuito, integrado |
| Template base | docusaurus-tailwind-shadcn-template | Já integra tudo |

## Links para visitar manualmente

| Site | URL | O que olhar |
|------|-----|-------------|
| Swarmpit | https://swarmpit.io | Hero com terminal, features grid numeradas, dark theme |
| Cetacean | https://cetacean.mazetti.me | Minimalismo, search Ctrl+K, demo link |
| Scribe | https://scribedocs.vercel.app | Callouts, code tabs, API endpoint component, cards |
| Verdaccio | https://verdaccio.org | OSS self-hosted, "used by" section, sponsor link |
| Quickwit | https://quickwit.io | Architecture diagram, testimonials, "open and free" section |
| Dyte | https://docs.dyte.io | Docs como landing page, SDK cards, community section |
| Docusaurus OpenAPI | https://docusaurus-openapi.tryingpan.dev | OpenAPI integration, color palette, breadcrumbs, tags |
| Docusaurus Showcase | https://docusaurus.io/showcase | Galeria completa de sites |
| Template Tailwind+shadcn | https://github.com/namnguyenthanhwork/docusaurus-tailwind-shadcn-template | Template base recomendado |
