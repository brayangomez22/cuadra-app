// Every query of imports starts with "imports".
export const importKeys = {
  all: ['imports'] as const,
  import: (id: string) => ['imports', id] as const,
  errors: (id: string, offset: number) => ['imports', id, 'errors', offset] as const,
};
