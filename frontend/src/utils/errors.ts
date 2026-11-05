import { toast } from 'vue-sonner'

export function describe(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export function reportError(error: unknown, title: string): void {
  console.error(`${title}:`, error)
  toast.error(title, { description: describe(error) })
}
