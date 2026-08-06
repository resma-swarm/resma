# RESMA CLI — Análise Detalhada UI/UX do k9s

## 1. Grid e Dimensões (Terminal 120x40)

O k9s usa um layout de proporções fixas para o header e dinâmico para o corpo.

### 1.1 Header (7 linhas fixas)
- **ClusterInfo (Esquerda - 50 colunas):** 
  - Linha 1: Contexto/Cluster (Bold Aqua)
  - Linha 2: User (Bold Cyan)
  - Linha 3: K8s Version (Bold White)
  - Linha 4: CPU % (Indicator bar)
  - Linha 5: MEM % (Indicator bar)
- **Menu Grid (Centro - Flexível):**
  - Grid de keybindings (max 6 linhas)
  - Formato: ` <key> desc `
  - Alinhamento: Colunas verticais
- **Logo/Status (Direita - 26 colunas):**
  - Logo ASCII (5 linhas)
  - Status Box (1 linha centralizada): ONLINE, COMMAND, FILTER, ERROR

### 1.2 Body (Área Flexível)
- **Tabela:** Ocupa 100% da largura e altura restante.
- **Paddings:** 1 char de respiro nas bordas laterais.
- **Selection:** Linha inteira com background colorido (não apenas o texto).
- **Header:** Linha fixa, cores contrastantes, indicadores de Sort (↑/↓).

### 1.3 Crumbs & Flash (2 linhas no rodapé)
- **Crumbs:** Chips estilo `<chip>` com backgrounds diferentes para histórico.
- **Flash:** Mensagens de status centralizadas com emojis.

---

## 2. Paleta de Cores e Estilos (Skin Default)

| Elemento | Cor (Hex/ANSI) | Estilo |
|----------|----------------|--------|
| Background Geral | #000000 | Solid |
| Foreground Geral | #A9A9A9 | Normal |
| Títulos de Seção | #00FFFF (Aqua) | Bold |
| Keybindings (Keys) | #00CED1 (DarkCyan) | Bold |
| Keybindings (Desc) | #808080 | Normal |
| Cursor da Tabela | Background #4B0082 | White text |
| Sucesso | #00FF00 | Bold |
| Erro/Crítico | #FF0000 | Bold |
| Warning | #FFA500 | Bold |

---

## 3. Comportamento da Tabela

O k9s utiliza `tview.Table`, mas simularemos com `bubbles/table` + auto-sizing:
1. **Vertical Fill:** A tabela deve preencher toda a altura entre o header e o crumb bar.
2. **Horizontal Fill:** As colunas devem se expandir para não deixar buracos pretos à direita.
3. **No Centralized Tables:** O wireframe anterior centralizou a tabela; o k9s alinha à esquerda com bordas que tocam os limites do terminal.
4. **Row Spacing:** No k9s, as linhas são densas (1 linha por registro), mas a legibilidade vem do contraste de cores e do cursor largo.

---

## 4. Evolução do Wireframe (PRÓXIMOS PASSOS)

Vou reescrever o `layout.go` e os componentes para:
- Implementar o **ClusterInfo** detalhado.
- Criar o **Menu Grid** real (não apenas uma lista).
- Fazer a **Tabela ocupar 100%** do espaço útil.
- Adicionar o **Logo ASCII de 5 linhas**.
- Implementar **Crumbs como chips** coloridos.
