import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null || isNaN(bytes)) return "0 B"
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export function formatCPU(cpu: number | undefined | null): string {
  if (cpu === undefined || cpu === null) return "0.00%"
  return `${cpu.toFixed(2)}%`
}

export function formatCores(cores: number | undefined | null): string {
  if (cores === undefined || cores === null || isNaN(cores)) return "0.00"
  return `${cores.toFixed(2)}`
}
