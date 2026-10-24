import type { components } from '@/shared/api/schema';

export type Module = components['schemas']['Module'];

/** What a module needs active first, by name: "Necesita: Variantes". An unknown one shows its id. */
export function moduleNeeds(module: Module, modules: readonly Module[]) {
  if (module.depends_on.length === 0) return undefined;
  const names = new Map(modules.map((m) => [m.id, m.name]));
  return `Necesita: ${module.depends_on.map((id) => names.get(id) ?? id).join(', ')}`;
}
