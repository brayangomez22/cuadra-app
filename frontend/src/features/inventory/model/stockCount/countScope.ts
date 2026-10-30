/** What a count covers: the location, or its categories by name ("Algunas categorías" until the names are known). */
export function countScope(categoryIds: string[], categories: { id: string; name: string }[] | undefined): string {
  if (categoryIds.length === 0) return 'Toda la sede';
  const names = (categories ?? []).filter((c) => categoryIds.includes(c.id)).map((c) => c.name);
  return names.length > 0 ? names.join(', ') : 'Algunas categorías';
}
