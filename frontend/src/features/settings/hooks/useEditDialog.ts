import { useCallback, useState } from 'react';

/** The create or edit dialog of a settings list: closed, adding a new item or editing one. */
export function useEditDialog<T>() {
  const [state, setState] = useState<{ item: T | null } | null>(null);

  const openNew = useCallback(() => setState({ item: null }), []);
  const openEdit = useCallback((item: T) => setState({ item }), []);
  const close = useCallback(() => setState(null), []);

  return { isOpen: state !== null, editing: state?.item ?? null, openNew, openEdit, close };
}
