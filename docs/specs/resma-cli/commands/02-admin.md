# RESMA CLI — Commands: Admin (Write)

> **Status:** Referência de comandos administrativos — 22 comandos write

> **Diretriz de naming:** Todos os comandos, subcomandos, flags, parâmetros e valores de flags são em **inglês americano** (ex: `--confirm`, `--dry-run`, `--service`, `services archive`). Prosa e descrições em **português brasileiro**.

> **Convenções de auth:**
> - **JWT** — `Authorization: Bearer <jwt>` contra `/api/*` (interno/UI). Obtenha o token via `resma auth login` (credenciais persistidas em `~/.config/resma/credentials.json`, XDG-compatible).
> - **API Key** — `Authorization: Bearer <api-key>` contra `/api/v1/*` (público, automação). Selecione com `--api-key`.
> - Todos os comandos deste documento usam **JWT** salvo indicação em contrário no campo **Auth**.
> - **RBAC** indica o papel mínimo exigido: `owner` ⊃ `admin` ⊃ `user`. Quando o campo traz `owner/admin`, `user` é recusado com `403 Forbidden`.
> - **R/W** indica a classificação de operação: `Read` (idempotente, sem efeito colateral) ou `Write` (muta estado). Todos os comandos aqui são `Write` salvo os listados como `Read` no campo **R/W** (apenas os subcomandos `list`/`preview` de grupos administrativos).
> - **`--confirm`** — Comandos destrutivos ou com efeito colateral significativo exigem a flag `--confirm` para executar. Sem ela, o CLI imprime o plano de ação e aborta com exit code `1`.

---

## Sumário

| # | Command | Endpoint | RBAC | --confirm | Descrição |
|---|---------|----------|------|-----------|-----------|
| 1 | `resma services archive <name>` | `PATCH /api/services/{name}/archive` | owner/admin | ✅ | Arquiva um serviço (soft delete) |
| 2 | `resma services restore <name>` | `PATCH /api/services/{name}/restore` | owner/admin | ✅ | Restaura um serviço arquivado |
| 3 | `resma recommendations recalculate [service]` | `POST /api/recommendations/recalculate` ou `POST /api/recommendations/{service}/recalculate` | any | — | Recalcula recomendações (todos ou um serviço) |
| 4 | `resma recommendations apply <service>` | `POST /api/recommendations/{service}/apply` | owner/admin | ✅ | Aplica a recomendação ativa de um serviço |
| 5 | `resma rollback-watches rollback <id>` | `POST /api/rollback-watches/{id}/rollback` | owner/admin | ✅ | Executa o rollback monitorado de um watch |
| 6 | `resma rollback-watches cancel <id>` | `POST /api/rollback-watches/{id}/cancel` | owner/admin | ✅ | Cancela um watch de rollback |
| 7 | `resma schedules create` | `POST /api/schedules` | owner/admin | — | Cria um agendamento de mudança de recursos |
| 8 | `resma schedules cancel <id>` | `DELETE /api/schedules/{id}` | owner/admin | ✅ | Cancela um agendamento |
| 9 | `resma templates create` | `POST /api/templates` | owner/admin | — | Cria um template de recursos |
| 10 | `resma templates update <id>` | `PUT /api/templates/{id}` | owner/admin | — | Atualiza um template |
| 11 | `resma templates delete <id>` | `DELETE /api/templates/{id}` | owner/admin | ✅ | Remove um template |
| 12 | `resma templates apply <name> <service>` | `POST /api/templates/{name}/apply/{service}` | owner/admin | ✅ | Aplica um template a um serviço |
| 13 | `resma users list` | `GET /api/users` | owner/admin | — | Lista usuários (admin-only) |
| 14 | `resma users create` | `POST /api/users` | owner/admin | — | Cria um usuário |
| 15 | `resma users update <id>` | `PATCH /api/users/{id}` | owner/admin | — | Atualiza o papel de um usuário |
| 16 | `resma users delete <id>` | `DELETE /api/users/{id}` | owner/admin | ✅ | Remove um usuário |
| 17 | `resma api-keys list` | `GET /api/auth/api-keys` | owner/admin | — | Lista API keys |
| 18 | `resma api-keys create` | `POST /api/auth/api-keys` | owner/admin | — | Cria uma API key |
| 19 | `resma api-keys revoke <id>` | `DELETE /api/auth/api-keys/{id}` | owner/admin | ✅ | Revoga uma API key |
| 20 | `resma api-keys update <id>` | `PATCH /api/auth/api-keys/{id}` | owner/admin | — | Atualiza o nome de uma API key |
| 21 | `resma settings list` | `GET /api/settings` | any | — | Lista configurações globais |
| 22 | `resma settings update` | `PUT /api/settings` | owner/admin | — | Atualiza configurações globais |
| 23 | `resma prune preview` | `GET /api/prune/preview` | owner/admin | — | Pré-visualiza o que seria removido |
| 24 | `resma prune services [--dry-run]` | `POST /api/prune/services-stale` | owner/admin | ✅ (exec) | Remove serviços stale |
| 25 | `resma prune nodes [--dry-run]` | `POST /api/prune/nodes-stale` | owner/admin | ✅ (exec) | Remove nodes stale |
| 26 | `resma prune tasks [--dry-run]` | `POST /api/prune/tasks-orphan` | owner/admin | ✅ (exec) | Remove tasks órfãs |
| 27 | `resma prune metrics [--dry-run]` | `POST /api/prune/metrics` | owner/admin | ✅ (exec) | Remove métricas antigas |
| 28 | `resma prune change-log [--dry-run]` | `POST /api/prune/change-log` | owner/admin | ✅ (exec) | Remove entradas antigas do change-log |
| 29 | `resma prune volume-metrics [--dry-run]` | `POST /api/prune/volume-metrics` | owner/admin | ✅ (exec) | Remove métricas de volume antigas |

> **Contagem:** 29 entradas na tabela (22 comandos write + 7 subcomandos read/preview de grupos administrativos). O título "22 comandos write" refere-se aos comandos que mutam estado — `list`/`preview` são `Read` listados por conveniência de agrupamento.

---

## 1. Services — Archive / Restore

### 1.1 `resma services archive <name>`

**Syntax**
```
resma services archive <name> [--confirm] [--output table|json|yaml]
```

**Descrição** — Arquiva (soft delete) um serviço do RESMA. O serviço deixa de aparecer em listagens padrão (`services list`) e na coleta ativa de métricas, mas seus dados históricos (métricas, change-log, recomendações) são preservados para auditoria. A operação é reversível via `services restore`.

**API**
- **Method:** `PATCH`
- **Path:** `/api/services/{name}/archive`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a execução. Sem ela, o CLI apenas imprime o plano e aborta. |
| `--output` | string | `table` | Formato de saída: `table`, `json` ou `yaml`. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o comando imprime o serviço alvo e o estado resultante (arquivado) e sai com código `1`.

**Exemplo de uso**
```bash
resma services archive nginx-web --confirm
```

**Exemplo de saída**
```
✓ Service "nginx-web" archived.
  State: active → archived
  Historical data preserved (metrics, change-log, recommendations).
  Restore with: resma services restore nginx-web
```

---

### 1.2 `resma services restore <name>`

**Syntax**
```
resma services restore <name> [--confirm] [--output table|json|yaml]
```

**Descrição** — Restaura um serviço previamente arquivado, devolvendo-o às listagens padrão e à coleta ativa de métricas. Útil quando um serviço foi arquivado por engano ou retorna ao cluster após manutenção.

**API**
- **Method:** `PATCH`
- **Path:** `/api/services/{name}/restore`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a execução. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem ela, o CLI mostra o serviço a ser restaurado e aborta.

**Exemplo de uso**
```bash
resma services restore nginx-web --confirm
```

**Exemplo de saída**
```
✓ Service "nginx-web" restored.
  State: archived → active
  Metric collection resumed.
```

---

## 2. Recommendations — Recalculate / Apply

### 2.1 `resma recommendations recalculate [service]`

**Syntax**
```
resma recommendations recalculate [service] [--output table|json|yaml]
```

**Descrição** — Dispara o recálculo de recomendações de limites de CPU/memória. Sem argumento, recalcula para **todos** os serviços com métricas suficientes; com `<service>`, recalcula apenas para o serviço indicado. A operação é assíncrona no backend (enfileirada no ML sidecar) — o CLI aguarda a conclusão e imprime o resumo.

**API**
- **Method:** `POST`
- **Path:** `POST /api/recommendations/recalculate` (todos) **ou** `POST /api/recommendations/{service}/recalculate` (um serviço)
- **Auth:** JWT
- **RBAC:** any (qualquer papel autenticado: owner/admin/user)
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--output` | string | `table` | Formato de saída. |
| `--wait` | bool | `true` | Aguarda a conclusão do recálculo antes de retornar. Use `--wait=false` para enfileirar e sair. |

**`--confirm`** — Não exigida. O recálculo é idempotente e não muta limites em produção (apenas regenera recomendações candidatas).

**Exemplo de uso — todos os serviços**
```bash
resma recommendations recalculate
```

**Exemplo de saída — todos**
```
✓ Recalculation queued for 14 services.
  Completed: 12  Skipped (insufficient data): 2

  SERVICE       CPU RECO    MEM RECO    STATUS
  nginx-web     250m        128Mi       updated
  api-gateway   500m        256Mi       updated
  db-postgres   1000m       1Gi         unchanged
  ...
```

**Exemplo de uso — um serviço**
```bash
resma recommendations recalculate api-gateway
```

**Exemplo de saída — um serviço**
```
✓ Recalculation completed for "api-gateway".
  CPU: 500m (was 250m)   Mem: 256Mi (was 128Mi)
  Confidence: 0.87   Samples: 1,440
```

---

### 2.2 `resma recommendations apply <service>`

**Syntax**
```
resma recommendations apply <service> [--confirm] [--output table|json|yaml]
```

**Descrição** — Aplica a recomendação ativa do serviço ao cluster, atualizando os limites de CPU/memória do serviço no Docker Swarm via Docker SDK. Cria um registro no change-log e, se configurado, um watch de rollback automático caso o serviço apresente instabilidade pós-aplicação.

**API**
- **Method:** `POST`
- **Path:** `/api/recommendations/{service}/apply`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a aplicação. |
| `--output` | string | `table` | Formato de saída. |
| `--watch` | bool | `true` | Cria um rollback-watch automático após aplicar. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o diff (limites atuais vs. recomendados) e aborta.

**Exemplo de uso**
```bash
resma recommendations apply api-gateway --confirm
```

**Exemplo de saída**
```
✓ Recommendation applied to "api-gateway".
  CPU: 250m → 500m
  Mem: 128Mi → 256Mi
  Change-log entry: #4821
  Rollback watch: #339 (auto, 30m window)
```

---

## 3. Rollback Watches — Rollback / Cancel

### 3.1 `resma rollback-watches rollback <id>`

**Syntax**
```
resma rollback-watches rollback <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Executa manualmente o rollback associado a um watch, revertendo os limites do serviço ao estado anterior ao apply que originou o watch. Útil quando o monitor automático detectou regressão mas o operador quer forçar o rollback imediato em vez de aguardar o timeout.

**API**
- **Method:** `POST`
- **Path:** `/api/rollback-watches/{id}/rollback`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma o rollback. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o watch e os limites que seriam restaurados, e aborta.

**Exemplo de uso**
```bash
resma rollback-watches rollback 339 --confirm
```

**Exemplo de saída**
```
✓ Rollback executed for watch #339.
  Service: api-gateway
  CPU: 500m → 250m   Mem: 256Mi → 128Mi
  Watch status: rolled-back
  Change-log entry: #4823
```

---

### 3.2 `resma rollback-watches cancel <id>`

**Syntax**
```
resma rollback-watches cancel <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Cancela um watch de rollback ativo, impedindo que o rollback automático dispare. Use quando o operador conclui que a nova configuração está estável e o watch não é mais necessário.

**API**
- **Method:** `POST`
- **Path:** `/api/rollback-watches/{id}/cancel`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma o cancelamento. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o watch e aborta.

**Exemplo de uso**
```bash
resma rollback-watches cancel 339 --confirm
```

**Exemplo de saída**
```
✓ Watch #339 cancelled.
  Service: api-gateway
  Status: active → cancelled
  No automatic rollback will fire.
```

---

## 4. Schedules — Create / Cancel

### 4.1 `resma schedules create`

**Syntax**
```
resma schedules create --service <name> [--cpu <q>] [--mem <q>] [--tier <t>] --at <datetime> [--confirm] [--output table|json|yaml]
```

**Descrição** — Cria um agendamento que aplica uma mudança de recursos (CPU/memória/tier) a um serviço em um horário futuro. O scheduler do RESMA executa a mudança no momento definido e registra no change-log.

**API**
- **Method:** `POST`
- **Path:** `/api/schedules`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--service` | string | **obrigatória** | Nome do serviço alvo. |
| `--cpu` | quantity | — | Novo limite de CPU (ex: `500m`, `1.5`). |
| `--mem` | quantity | — | Novo limite de memória (ex: `256Mi`, `1Gi`). |
| `--tier` | string | — | Tier de recursos a aplicar (ex: `small`, `medium`, `large`). |
| `--at` | datetime | **obrigatória** | Momento de execução (RFC3339 ou relativo: `2026-01-15T14:30:00Z`, `in 2h`). |
| `--confirm` | bool | `false` | Confirma a criação. |
| `--output` | string | `table` | Formato de saída. |

> Ao menos uma de `--cpu`, `--mem` ou `--tier` deve estar presente.

**`--confirm`** — Recomendada. Sem `--confirm`, o CLI imprime o plano (serviço, mudança, horário) e pede confirmação interativa; em modo não-interativo, aborta com código `1`.

**Exemplo de uso**
```bash
resma schedules create --service api-gateway --cpu 750m --mem 384Mi --at "2026-01-15T14:30:00Z" --confirm
```

**Exemplo de saída**
```
✓ Schedule #128 created.
  Service:  api-gateway
  CPU:      → 750m
  Mem:      → 384Mi
  At:       2026-01-15T14:30:00Z
  Cancel with: resma schedules cancel 128
```

---

### 4.2 `resma schedules cancel <id>`

**Syntax**
```
resma schedules cancel <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Cancela um agendamento pendente. Agendamentos já executados não podem ser cancelados (use o change-log para reverter).

**API**
- **Method:** `DELETE`
- **Path:** `/api/schedules/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma o cancelamento. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o agendamento e aborta.

**Exemplo de uso**
```bash
resma schedules cancel 128 --confirm
```

**Exemplo de saída**
```
✓ Schedule #128 cancelled.
  Service: api-gateway
  Status:  pending → cancelled
```

---

## 5. Templates — Create / Update / Delete / Apply

### 5.1 `resma templates create`

**Syntax**
```
resma templates create --name <name> [--cpu <q>] [--mem <q>] --stacks <csv> [--output table|json|yaml]
```

**Descrição** — Cria um template de recursos reutilizável. Templates agrupam limites de CPU/memória e um conjunto de stacks às quais se aplicam, permitindo padronizar configurações entre serviços equivalentes.

**API**
- **Method:** `POST`
- **Path:** `/api/templates`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--name` | string | **obrigatória** | Nome único do template. |
| `--cpu` | quantity | — | Limite de CPU do template. |
| `--mem` | quantity | — | Limite de memória do template. |
| `--stacks` | csv | **obrigatória** | Lista de stacks (ex: `web,api,edge`). |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida (operação de criação não-destrutiva).

**Exemplo de uso**
```bash
resma templates create --name web-small --cpu 250m --mem 128Mi --stacks web,edge
```

**Exemplo de saída**
```
✓ Template "web-small" created.
  CPU:     250m
  Mem:     128Mi
  Stacks:  web, edge
  Apply with: resma templates apply web-small <service>
```

---

### 5.2 `resma templates update <id>`

**Syntax**
```
resma templates update <id> [--name <name>] [--cpu <q>] [--mem <q>] [--stacks <csv>] [--output table|json|yaml]
```

**Descrição** — Atualiza campos de um template existente. Apenas os campos fornecidos via flags são sobrescritos (PATCH semântico).

**API**
- **Method:** `PUT`
- **Path:** `/api/templates/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--name` | string | — | Novo nome. |
| `--cpu` | quantity | — | Novo limite de CPU. |
| `--mem` | quantity | — | Novo limite de memória. |
| `--stacks` | csv | — | Novo conjunto de stacks. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida.

**Exemplo de uso**
```bash
resma templates update 7 --cpu 500m --mem 256Mi
```

**Exemplo de saída**
```
✓ Template #7 updated.
  CPU:  250m → 500m
  Mem:  128Mi → 256Mi
```

---

### 5.3 `resma templates delete <id>`

**Syntax**
```
resma templates delete <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove permanentemente um template. Serviços que já tiveram o template aplicado não são afetados — apenas a definição reutilizável deixa de existir.

**API**
- **Method:** `DELETE`
- **Path:** `/api/templates/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a remoção. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o template e aborta.

**Exemplo de uso**
```bash
resma templates delete 7 --confirm
```

**Exemplo de saída**
```
✓ Template #7 deleted.
  Name: web-small
  Applied services are unaffected.
```

---

### 5.4 `resma templates apply <name> <service>`

**Syntax**
```
resma templates apply <name> <service> [--confirm] [--output table|json|yaml]
```

**Descrição** — Aplica um template (identificado por nome) a um serviço, sobrescrevendo seus limites de CPU/memória no cluster. Registra a mudança no change-log e cria um rollback-watch se configurado.

**API**
- **Method:** `POST`
- **Path:** `/api/templates/{name}/apply/{service}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a aplicação. |
| `--output` | string | `table` | Formato de saída. |
| `--watch` | bool | `true` | Cria um rollback-watch automático. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o diff (limites atuais vs. template) e aborta.

**Exemplo de uso**
```bash
resma templates apply web-small nginx-web --confirm
```

**Exemplo de saída**
```
✓ Template "web-small" applied to "nginx-web".
  CPU: 500m → 250m
  Mem: 256Mi → 128Mi
  Change-log entry: #4825
  Rollback watch: #341 (auto, 30m window)
```

---

## 6. Users — List / Create / Update / Delete

### 6.1 `resma users list`

**Syntax**
```
resma users list [--output table|json|yaml]
```

**Descrição** — Lista todos os usuários do RESMA. Acesso administrativo: mesmo sendo uma operação de leitura, o endpoint é restrito a owner/admin por expor identidades e papéis.

**API**
- **Method:** `GET`
- **Path:** `/api/users`
- **Auth:** JWT
- **RBAC:** owner/admin (read com acesso admin-only)
- **R/W:** Read

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida (operação de leitura).

**Exemplo de uso**
```bash
resma users list
```

**Exemplo de saída**
```
ID  USERNAME       ROLE     CREATED
1   admin          owner    2025-11-02
2   ops-alice      admin    2025-11-05
3   dev-bob        user     2025-12-10
```

---

### 6.2 `resma users create`

**Syntax**
```
resma users create --username <name> --password <pass> --role <admin|user> [--output table|json|yaml]
```

**Descrição** — Cria um novo usuário. A senha é hasheada com bcrypt no backend. O papel `owner` é reservado ao fluxo de onboarding (primeiro usuário) e não pode ser atribuído por este comando.

**API**
- **Method:** `POST`
- **Path:** `/api/users`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--username` | string | **obrigatória** | Nome de usuário único. |
| `--password` | string | **obrigatória** | Senha (lida da flag, ou via prompt interativo se omitida). |
| `--role` | string | `user` | Papel: `admin` ou `user`. `owner` é recusado. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida (operação de criação não-destrutiva).

**Exemplo de uso**
```bash
resma users create --username ops-carol --role admin
# Password: ******** (prompt interativo)
```

**Exemplo de saída**
```
✓ User "ops-carol" created.
  ID:    4
  Role:  admin
```

---

### 6.3 `resma users update <id>`

**Syntax**
```
resma users update <id> --role <admin|user> [--output table|json|yaml]
```

**Descrição** — Atualiza o papel de um usuário. Não é possível promover a `owner` nem rebaixar o último `owner` restante (proteção contra lockout administrativo).

**API**
- **Method:** `PATCH`
- **Path:** `/api/users/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--role` | string | — | Novo papel: `admin` ou `user`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida (a API valida proteções de owner).

**Exemplo de uso**
```bash
resma users update 3 --role admin
```

**Exemplo de saída**
```
✓ User #3 updated.
  Role: user → admin
```

---

### 6.4 `resma users delete <id>`

**Syntax**
```
resma users delete <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove um usuário. O backend recusa remover o último `owner` para evitar perda de acesso administrativo. Sessões JWT ativas do usuário removido são invalidadas.

**API**
- **Method:** `DELETE`
- **Path:** `/api/users/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a remoção. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra o usuário e aborta.

**Exemplo de uso**
```bash
resma users delete 3 --confirm
```

**Exemplo de saída**
```
✓ User #3 deleted.
  Username: dev-bob
  Active sessions invalidated.
```

---

## 7. API Keys — List / Create / Revoke / Update

### 7.1 `resma api-keys list`

**Syntax**
```
resma api-keys list [--output table|json|yaml]
```

**Descrição** — Lista as API keys do RESMA. Por segurança, o valor da chave **não** é exibido — apenas o nome, scopes e indicador de status ativo/revogado.

**API**
- **Method:** `GET`
- **Path:** `/api/auth/api-keys`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Read

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida.

**Exemplo de uso**
```bash
resma api-keys list
```

**Exemplo de saída**
```
ID  NAME             SCOPES                STATUS     CREATED
1   ci-deploy        services:read         active     2025-11-02
2   grafana-readonly metrics:read          active     2025-12-01
3   legacy-key       —                     revoked    2025-10-15
```

---

### 7.2 `resma api-keys create`

**Syntax**
```
resma api-keys create --name <name> --scopes <csv> [--output table|json|yaml]
```

**Descrição** — Cria uma nova API key com os scopes indicados. O valor da chave é exibido **uma única vez** no momento da criação — não é recuperável depois.

**API**
- **Method:** `POST`
- **Path:** `/api/auth/api-keys`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--name` | string | **obrigatória** | Nome identificador da chave. |
| `--scopes` | csv | **obrigatória** | Scopes (ex: `services:read,metrics:read,recommendations:write`). |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida.

**Exemplo de uso**
```bash
resma api-keys create --name ci-deploy --scopes services:read,metrics:read
```

**Exemplo de saída**
```
✓ API key "ci-deploy" created.
  ID:      4
  Scopes:  services:read, metrics:read

  ⚠️  Key (shown once):
      rsk_9f8e7d6c5b4a3210_fedcba9876543210

  Store it securely — it cannot be retrieved again.
```

---

### 7.3 `resma api-keys revoke <id>`

**Syntax**
```
resma api-keys revoke <id> [--confirm] [--output table|json|yaml]
```

**Descrição** — Revoga permanentemente uma API key. Requisições futuras usando a chave revogada recebem `401 Unauthorized`.

**API**
- **Method:** `DELETE`
- **Path:** `/api/auth/api-keys/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--confirm` | bool | `false` | Confirma a revogação. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória. Sem `--confirm`, o CLI mostra a chave e aborta.

**Exemplo de uso**
```bash
resma api-keys revoke 4 --confirm
```

**Exemplo de saída**
```
✓ API key #4 revoked.
  Name: ci-deploy
  Status: active → revoked
  Future requests with this key will be rejected (401).
```

---

### 7.4 `resma api-keys update <id>`

**Syntax**
```
resma api-keys update <id> --name <name> [--output table|json|yaml]
```

**Descrição** — Atualiza o nome identificador de uma API key. Os scopes e o valor da chave não são alterados por este comando.

**API**
- **Method:** `PATCH`
- **Path:** `/api/auth/api-keys/{id}`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--name` | string | — | Novo nome. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida.

**Exemplo de uso**
```bash
resma api-keys update 4 --name ci-deploy-v2
```

**Exemplo de saída**
```
✓ API key #4 updated.
  Name: ci-deploy → ci-deploy-v2
```

---

## 8. Settings — List / Update

### 8.1 `resma settings list`

**Syntax**
```
resma settings list [--output table|json|yaml]
```

**Descrição** — Lista as configurações globais do RESMA (intervalo de coleta, retenção de métricas, threshold de outliers, etc.). Qualquer usuário autenticado pode consultar.

**API**
- **Method:** `GET`
- **Path:** `/api/settings`
- **Auth:** JWT
- **RBAC:** any (qualquer papel autenticado)
- **R/W:** Read

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não exigida.

**Exemplo de uso**
```bash
resma settings list
```

**Exemplo de saída**
```
KEY                  VALUE       DESCRIPTION
collect_interval     30s         Intervalo de coleta de métricas
retention_days       90          Dias de retenção de métricas
outlier_threshold    3.0         Desvios-padrão para detecção de outliers
rollback_window      30m         Janela de observação pós-apply
auto_rollback        true        Rollback automático em regressão
```

---

### 8.2 `resma settings update`

**Syntax**
```
resma settings update [--collect-interval <duration>] [--retention-days <int>] [--outlier-threshold <float>] [...] [--output table|json|yaml]
```

**Descrição** — Atualiza configurações globais do RESMA. Apenas os campos fornecidos via flags são sobrescritos. Mudanças em `collect_interval` e `retention_days` entram em vigor no próximo ciclo do scheduler.

**API**
- **Method:** `PUT`
- **Path:** `/api/settings`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--collect-interval` | duration | — | Intervalo de coleta (ex: `30s`, `1m`). |
| `--retention-days` | int | — | Dias de retenção de métricas. |
| `--outlier-threshold` | float | — | Threshold em desvios-padrão para outliers. |
| `--rollback-window` | duration | — | Janela de observação pós-apply. |
| `--auto-rollback` | bool | — | Habilita/desabilita rollback automático. |
| `--output` | string | `table` | Formato de saída. |

> Outras flags de configuração podem existir conforme o schema de settings do backend; o CLI repassa qualquer flag `--<key>` reconhecida.

**`--confirm`** — Não exigida (operação de configuração, não-destrutiva).

**Exemplo de uso**
```bash
resma settings update --collect-interval 15s --retention-days 180
```

**Exemplo de saída**
```
✓ Settings updated.
  collect_interval:  30s  → 15s
  retention_days:    90   → 180
  Effective on next scheduler cycle.
```

---

## 9. Prune — Preview / Services / Nodes / Tasks / Metrics / Change-Log / Volume-Metrics

> **Convenção de prune:** Todos os subcomandos `prune` operam em modo **dry-run por padrão** (`--dry-run=true`). Para executar a remoção de fato, passe `--dry-run=false` **e** `--confirm`. O subcomando `prune preview` é sempre read-only e não possui `--confirm`.

### 9.1 `resma prune preview`

**Syntax**
```
resma prune preview [--output table|json|yaml]
```

**Descrição** — Pré-visualiza, em uma única consulta, tudo que seria removido por todos os subcomandos de prune (serviços stale, nodes stale, tasks órfãs, métricas antigas, change-log antigo, métricas de volume antigas). Operação read-only — não modifica dados.

**API**
- **Method:** `GET`
- **Path:** `/api/prune/preview`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Read

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Não aplicável (read-only).

**Exemplo de uso**
```bash
resma prune preview
```

**Exemplo de saída**
```
PRUNE PREVIEW (dry-run, no changes will be made)

  Category         Count   Size est.   Oldest
  services-stale   3       —           2025-09-01
  nodes-stale      1       —           2025-08-20
  tasks-orphan     42      —           2025-10-10
  metrics          1.2M    480 MB      2025-07-01
  change-log       318     12 MB       2025-06-15
  volume-metrics   86K     34 MB       2025-07-01

  Total reclaimable: ~526 MB
  Run a specific prune with --dry-run=false --confirm to execute.
```

---

### 9.2 `resma prune services [--dry-run]`

**Syntax**
```
resma prune services [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove serviços marcados como stale (arquivados há mais tempo que o limite de retenção e sem métricas recentes). Em dry-run (padrão), apenas lista o que seria removido.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/services-stale`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Quando `true`, apenas simula. Quando `false`, executa a remoção. |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar (`--dry-run=false --confirm`). Sem ela, mesmo com `--dry-run=false`, o CLI aborta.

**Exemplo de uso — dry-run (padrão)**
```bash
resma prune services
```

**Exemplo de saída — dry-run**
```
PRUNE services-stale (dry-run — no changes made)

  SERVICE          ARCHIVED     LAST METRIC    ACTION
  legacy-cache     2025-08-01   2025-08-15     would delete
  old-worker       2025-08-20   2025-09-01     would delete
  stale-sidecar    2025-09-01   2025-09-10     would delete

  3 services would be removed.
  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune services --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE services-stale (executing)

  ✓ legacy-cache     deleted
  ✓ old-worker       deleted
  ✓ stale-sidecar    deleted

  3 services removed.
  Change-log entries: #4826, #4827, #4828
```

---

### 9.3 `resma prune nodes [--dry-run]`

**Syntax**
```
resma prune nodes [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove nodes stale (nós que saíram do Swarm e sem atividade recente). Em dry-run (padrão), apenas lista.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/nodes-stale`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Simula (`true`) ou executa (`false`). |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar.

**Exemplo de uso — dry-run**
```bash
resma prune nodes
```

**Exemplo de saída — dry-run**
```
PRUNE nodes-stale (dry-run — no changes made)

  NODE ID            HOSTNAME        LAST SEEN     ACTION
  n7a2b...           worker-03       2025-08-20    would delete
  n9c4d...           worker-07       2025-09-05    would delete

  2 nodes would be removed.
  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune nodes --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE nodes-stale (executing)

  ✓ n7a2b...  worker-03   deleted
  ✓ n9c4d...  worker-07   deleted

  2 nodes removed.
  Change-log entries: #4829, #4830
```

---

### 9.4 `resma prune tasks [--dry-run]`

**Syntax**
```
resma prune tasks [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove tasks órfãs (tasks sem serviço pai ou em estado terminal há mais tempo que o limite). Em dry-run (padrão), apenas lista.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/tasks-orphan`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Simula (`true`) ou executa (`false`). |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar.

**Exemplo de uso — dry-run**
```bash
resma prune tasks
```

**Exemplo de saída — dry-run**
```
PRUNE tasks-orphan (dry-run — no changes made)

  TASK ID            SERVICE         STATE       FINISHED       ACTION
  t1a2b...           legacy-cache   shutdown    2025-09-01     would delete
  t3c4d...           legacy-cache   failed      2025-09-02     would delete
  ... (40 more)

  42 tasks would be removed.
  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune tasks --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE tasks-orphan (executing)

  ✓ 42 tasks removed.
  Change-log entry: #4831
```

---

### 9.5 `resma prune metrics [--dry-run]`

**Syntax**
```
resma prune metrics [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove métricas antigas além do período de retenção configurado em `retention_days`. Em dry-run (padrão), estima o volume a ser removido.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/metrics`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Simula (`true`) ou executa (`false`). |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar.

**Exemplo de uso — dry-run**
```bash
resma prune metrics
```

**Exemplo de saída — dry-run**
```
PRUNE metrics (dry-run — no changes made)

  Retention: 90 days   Cutoff: 2025-10-18
  Rows to delete: 1,204,318   Size est.: ~480 MB
  Oldest row: 2025-07-01

  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune metrics --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE metrics (executing)

  ✓ 1,204,318 rows deleted.
  Reclaimed: ~480 MB
  Duration: 3.2s
  Change-log entry: #4832
```

---

### 9.6 `resma prune change-log [--dry-run]`

**Syntax**
```
resma prune change-log [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove entradas antigas do change-log além do período de retenção. Em dry-run (padrão), estima o volume.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/change-log`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Simula (`true`) ou executa (`false`). |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar.

**Exemplo de uso — dry-run**
```bash
resma prune change-log
```

**Exemplo de saída — dry-run**
```
PRUNE change-log (dry-run — no changes made)

  Retention: 90 days   Cutoff: 2025-10-18
  Entries to delete: 318   Size est.: ~12 MB
  Oldest entry: 2025-06-15

  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune change-log --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE change-log (executing)

  ✓ 318 entries deleted.
  Reclaimed: ~12 MB
  Change-log entry: #4833 (meta)
```

---

### 9.7 `resma prune volume-metrics [--dry-run]`

**Syntax**
```
resma prune volume-metrics [--dry-run=true|false] [--confirm] [--output table|json|yaml]
```

**Descrição** — Remove métricas de volume antigas além do período de retenção. Em dry-run (padrão), estima o volume.

**API**
- **Method:** `POST`
- **Path:** `/api/prune/volume-metrics`
- **Auth:** JWT
- **RBAC:** owner/admin
- **R/W:** Write

**Flags específicas**
| Flag | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `--dry-run` | bool | `true` | Simula (`true`) ou executa (`false`). |
| `--confirm` | bool | `false` | Obrigatória quando `--dry-run=false`. |
| `--output` | string | `table` | Formato de saída. |

**`--confirm`** — Obrigatória para executar.

**Exemplo de uso — dry-run**
```bash
resma prune volume-metrics
```

**Exemplo de saída — dry-run**
```
PRUNE volume-metrics (dry-run — no changes made)

  Retention: 90 days   Cutoff: 2025-10-18
  Rows to delete: 86,402   Size est.: ~34 MB
  Oldest row: 2025-07-01

  Run with --dry-run=false --confirm to execute.
```

**Exemplo de uso — execução**
```bash
resma prune volume-metrics --dry-run=false --confirm
```

**Exemplo de saída — execução**
```
PRUNE volume-metrics (executing)

  ✓ 86,402 rows deleted.
  Reclaimed: ~34 MB
  Duration: 0.8s
  Change-log entry: #4834
```

---

## 10. Notas Finais

### 10.1 Padrões de segurança

- **`--confirm` obrigatória** — Aplicada a todos os comandos destrutivos ou com efeito colateral irreversível (archive, restore, apply, rollback, cancel de schedule/watch, delete, revoke). Sem a flag, o CLI imprime o plano de ação e sai com código `1`.
- **`--dry-run` default `true`** — Aplicada a todos os subcomandos `prune`. A execução real exige `--dry-run=false` **e** `--confirm`.
- **RBAC owner/admin** — Comandos que mutam estado do cluster ou identidades exigem papel administrativo. O backend valida independentemente; o CLI apenas informa o requisito.
- **Proteção de owner** — `users create` recusa `--role owner`; `users update` e `users delete` protegem o último owner restante.

### 10.2 Formatos de saída

Todos os comandos aceitam `--output table|json|yaml`. Em `json`/`yaml`, a saída inclui o payload bruto da resposta da API — útil para automação e piping (`resma users list --output json | jq ...`).

### 10.3 Exit codes

| Código | Significado |
|--------|-------------|
| `0` | Sucesso. |
| `1` | Operação abortada (faltou `--confirm`, ou `--dry-run` ativo). |
| `2` | Erro de validação de flags/argumentos. |
| `3` | Erro de auth (JWT ausente/expirado, RBAC insuficiente). |
| `4` | Erro de rede/API (timeout, 5xx, conexão recusada). |
