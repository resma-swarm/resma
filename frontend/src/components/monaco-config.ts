import { configureMonacoYaml } from "monaco-yaml"

const templateSchema = {
  type: "object",
  title: "RESMA Template",
  description: "Configuração de recursos para serviços Docker Swarm",
  properties: {
    limits: {
      type: "object",
      description: "Limites máximos de recursos",
      properties: {
        cpus: { type: "string", description: "Limite de CPU em cores (ex: '0.50', '1.0')" },
        memory: { type: "string", description: "Limite de memória (ex: '512M', '1G', '256K')" },
      },
    },
    reservations: {
      type: "object",
      description: "Reserva garantida de recursos",
      properties: {
        cpus: { type: "string", description: "Reserva de CPU em cores (ex: '0.25', '0.5')" },
        memory: { type: "string", description: "Reserva de memória (ex: '256M', '512M')" },
      },
    },
    mem_margin: {
      type: "number",
      description: "Multiplicador de margem de memória sobre P95 (ex: 1.5 = 150% do P95)",
      minimum: 1.0,
      maximum: 3.0,
      default: 1.5,
    },
    cpu_margin: {
      type: "number",
      description: "Multiplicador de margem de CPU sobre P95 (ex: 1.5 = 150% do P95)",
      minimum: 1.0,
      maximum: 3.0,
      default: 1.5,
    },
    reservation_ratio: {
      type: "number",
      description: "Ratio de reservation sobre P50 (0.0 a 1.0, ex: 0.75 = 75% do P50)",
      minimum: 0.0,
      maximum: 1.0,
      default: 0.75,
    },
    leak_tolerance: {
      type: "number",
      description: "Tolerância para memory leak (1.0 = normal, >1 = mais tolerante)",
      minimum: 0.0,
      maximum: 2.0,
      default: 1.0,
    },
  },
}

export function setupMonacoYaml(monacoInstance: typeof import("monaco-editor")) {
  configureMonacoYaml(monacoInstance, {
    schemas: [
      {
        uri: "https://resma.local/template-schema.json",
        fileMatch: ["*"],
        schema: templateSchema,
      },
    ],
  })
}
