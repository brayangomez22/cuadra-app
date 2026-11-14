import type { Municipality } from '../../api/types';

type Option = { value: string; label: string };

/** Each department of the DANE's catalog once, in the catalog's order (by name). */
export function departmentOptions(list: Municipality[]): Option[] {
  const seen = new Map<string, string>();
  for (const m of list) seen.set(m.department_code, m.department_name);
  return [...seen].map(([value, label]) => ({ value, label }));
}

/** The municipalities of a department, in the catalog's order; none without a department. */
export function municipalityOptions(list: Municipality[], department: string): Option[] {
  return list.filter((m) => m.department_code === department).map((m) => ({ value: m.code, label: m.name }));
}

/** A DANE municipality's department: the first two digits of its code. */
export function departmentOf(code: string) {
  return code.slice(0, 2);
}
