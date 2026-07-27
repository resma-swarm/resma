# RESMA — Passos para Publicação no GitHub

> Este arquivo documenta os passos manuais necessários para publicar o RESMA no GitHub.
> Execute-os quando estiver pronto para tornar o projeto público.

## 1. Criar repositório no GitHub

```bash
# Opção A: via gh CLI
gh repo create resma --public --description "RESource MAnager for Docker Swarm"

# Opção B: via interface web
# Acesse https://github.com/new e crie um repo chamado "resma"
```

## 2. Substituir placeholders USER/user pelo seu username

Antes de commitar, substitua todos os placeholders `USER` e `user` pelo seu
username real do GitHub. Os arquivos que contêm esses placeholders:

```bash
# Listar arquivos com placeholders
grep -rn "ghcr.io/USER\|ghcr.io/user\|USER/resma\|user/resma" --include="*.yml" --include="*.yaml" --include="*.sh" --include="*.md" --include="*.ts" --include="*.go" .

# Substituir (exemplo com sed — ajuste SEU_USERNAME)
SEU_USERNAME="seu-username"
sed -i "s|ghcr.io/USER|ghcr.io/${SEU_USERNAME}|g" docker-stack.yml install.sh
sed -i "s|ghcr.io/user|ghcr.io/${SEU_USERNAME}|g" docker-stack.yml install.sh
sed -i "s|USER/resma|${SEU_USERNAME}/resma|g" install.sh README.md docs-site/docusaurus.config.ts
```

Arquivos a verificar:
- `docker-stack.yml` — `ghcr.io/${RESMA_REGISTRY:-user}/resma-api`
- `install.sh` — `IMAGE_PREFIX="ghcr.io/user"` e URL do one-liner
- `README.md` — URL do one-liner
- `docs-site/docusaurus.config.ts` — `url: 'https://USER.github.io'`
- `docs/benchmarks.md` — referências a GHCR

## 3. Configurar GitHub Pages

1. No repo: Settings → Pages → Source: **GitHub Actions**
2. O workflow `.github/workflows/docs.yml` fará o deploy automático do Docusaurus

## 4. Configurar secrets do CI (opcional)

O CI usa `GITHUB_TOKEN` (automático) para push para GHCR. Não precisa de secrets adicionais.

Se quiser deploy de docs em domínio próprio:
- Settings → Pages → Custom domain → adicionar `resma.seudominio.com`

## 5. Adicionar remote e fazer push

```bash
git remote add origin git@github.com:SEU_USERNAME/resma.git
git branch -M main
git push -u origin main
```

## 6. Criar primeira release

```bash
git tag v0.1.0
git push origin v0.1.0
```

O workflow `.github/workflows/release.yml` fará:
- Build multi-arch (amd64 + arm64) das 2 imagens (API + ML)
- Push para GHCR com tags `latest`, `v0.1.0`
- Criar GitHub Release com notes auto-geradas

## 7. Verificar CI

Após o push, verifique:
- Actions tab → workflow CI deve passar (go-lint, go-test, go-vet, docker-build)
- Packages tab → imagens `resma-api` e `resma-ml` devem aparecer
- Pages → site de docs deve estar acessível em `https://SEU_USERNAME.github.io/resma/`

## 8. Configurar branch protection (recomendado)

Settings → Branches → Add rule:
- Branch: `main`
- Require status checks: CI must pass
- Require pull request reviews: 1 approval
- Require linear history

## Issues conhecidos a resolver antes da publicação

1. **ML sidecar DuckDB lock** — o ML sidecar Python não consegue abrir o DuckDB
   em read-only enquanto o Go API tem lock exclusivo. Solução: o Go API deve
   proxyar as queries DB para o ML sidecar via HTTP, ou usar DuckDB HTTP extension.

2. **Tamanho da imagem API** — 189MB (target era <150MB). Para reduzir:
   - Usar `scratch` ou `alpine` no runtime stage (precisa de libc compatível)
   - Ou usar `distroless` com libc

3. **Testes Go** — coverage atual: config 85%, SSE 37.8%. Adicionar testes para
   auth, handlers, db, docker (com mocks).
