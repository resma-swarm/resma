# RESMA CLI — Evolução do Wireframe TUI

> **Status:** Wireframe de Alta Fidelidade Refinado (Suporte completo a Abas + Seleção de Linha de 100% + Espaçamento Respirável + Identidade RESMA)
> **Comando:** `go run ./cmd/wire-tui`
> **Compilação:** `go build -o wire-tui.exe ./cmd/wire-tui/`

## Melhorias Implementadas

1. **Régua Visual de Abas Explícita:**
   - Adicionada a barra de abas estilo `[1] Services  [2] Nodes  [3] Agents  [4] Tasks  [5] Alerts  [6] Recs`.
   - Compatibilidade mantida com teclas numéricas `1-6` e rotação via `Tab`/`Shift+Tab`.

2. **Preenchimento do Cursor 100% (Full Row Selection):**
   - A linha selecionada na tabela agora preenche **100% da largura da janela** com fundo Indigo (`#4B0082`) e texto em branco em negrito.
   - Corrigido o bug onde status como `stopped` ou `failed` quebravam o preenchimento por conta de escapes ANSI anteriores.

3. **Respiro e Espaçamento Vertical:**
   - Adicionado padding/respiro vertical entre as linhas da tabela (`\n\n`), dobrando a legibilidade de listas longas.

4. **Identidade Própria RESMA Dashboard:**
   - Refatoração dos namespaces e estilos internos (remoção de prefixos terceiros para nomes próprios do ecossistema RESMA).
   - Identidade com Swarm, Stacks, Role Manager e métricas do cluster agregadas.

## Como Executar

```bash
cd D:\allt\resma\app\cli
go run ./cmd/wire-tui
```
