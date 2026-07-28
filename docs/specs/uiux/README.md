# RESMA Docs-Site Redesign — Specs UI/UX

> Plano de redesign da landing page do docs-site (Docusaurus) para um visual
> dark premium inspirado em Linear/Supabase/shadcn. Specs detalhadas em
> `NN-*.md`. Execute uma task por chat, seguindo o AI Engineering Framework.
>
> **✅ Redesign completo (T1-T11)** — todas as 11 tasks concluídas. Landing
> page dark premium com hero split + terminal + aurora, bento grid de features,
> dashboard mockup, code example, comparison table, final CTA com social proof,
> efeitos visuais (border beam, glowing card, dotted background), scroll
> animations (Framer Motion LazyMotion + typewriter), SEO/OG image e passada
> final de performance/a11y (WCAG AA, prefers-reduced-motion, focus-visible).

## Status de implementação

| Task | Título | Esforço | Pré-requisitos | Status | Commit |
|------|--------|---------|----------------|--------|--------|
| T1 | Foundation: Tailwind + Paleta + Tipografia | Médio | Nenhum | ✅ Concluído | 0c26832 |
| T2 | Hero Redesign: Split + Terminal + Aurora + Badges | Alto | T1 | ✅ Concluído | 6874c48 |
| T3 | Feature Cards (3-up grid) | Médio | T1 | ✅ Concluído | e5bc9f2 |
| T4 | Install Command (bloco copiável) | Médio | T1 | ✅ Concluído | 4594131 |
| T5 | Dashboard Mockup (iframe/screenshot) | Médio | T1, T2 | ✅ Concluído | d500402 |
| T6 | Code Example (tabs Go/Python/YAML) | Médio | T1 | ✅ Concluído | 4199643 |
| T7 | Comparison Table (RESMA vs outros) | Baixo | T1 | ✅ Concluído | 71e1a3c |
| T8 | Visual Effects (border beam, glow) | Médio | T1, T2 | ✅ Concluído | 7f556a2 |
| T9 | Scroll Animations (reveal, typewriter) | Médio | T2, T3 | ✅ Concluído | 8ef51b7 |
| T10 | Final CTA + SEO (meta tags, OG) | Médio | T1, T2 | ✅ Concluído | bd1321d |
| T11 | Performance + A11y (audit final) | Médio | T8, T9 | ✅ Concluído | eddd48b |

## Convenções

- **Status:** ⬜ Pendente · 🔄 Em andamento · ✅ Concluído · ⛔ Bloqueado
- **Uma task por chat** — não execute múltiplas tasks na mesma sessão.
- **Pré-requisitos:** confirme que todos estão ✅ antes de iniciar. Se algum
  não estiver, PARE e reporte.
- **Commit:** `feat(uiux-TN): descrição` (local, sem push). Use PowerShell `-F`
  para mensagens multi-linha.
- **Validação:** Playwright MCP em http://localhost:3001/ (snapshot, console
  errors, browser_find, resize 375px e 1280px).
- **Backend Python em `backend/` é referência — não modificar.**
- **Frontend app em `frontend/` só é tocado em 0b.11** — esta série de tasks
  afeta apenas `docs-site/`.

## Ordem recomendada

T1 → T2 → (T3, T4, T6, T7 paralelo) → T5 → T8 → T9 → T10 → T11
