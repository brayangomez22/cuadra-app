import { useState } from 'react';

/**
 * The item a settings list asks to confirm before deleting it, and the props of
 * its ConfirmDeleteDialog. A failed delete rejects, so the dialog shows why.
 */
export function useDeleteConfirmation<T>(remove: (item: T) => Promise<unknown>) {
  const [target, setTarget] = useState<T | null>(null);

  return {
    target,
    ask: setTarget,
    dialogProps: {
      open: target !== null,
      onOpenChange: (open: boolean) => {
        if (!open) setTarget(null);
      },
      onConfirm: async () => {
        if (target !== null) await remove(target);
      },
    },
  };
}
