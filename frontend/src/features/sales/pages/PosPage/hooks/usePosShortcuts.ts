import { useEffect, type RefObject } from 'react';

type Options = {
  search: RefObject<HTMLInputElement | null>;
  /** Whether a dialog is open: it handles its own keys (Escape closes it). */
  dialogOpen: boolean;
  canCharge: boolean;
  hasLines: boolean;
  onCharge: () => void;
  onCancel: () => void;
  /** Opens the choice of the sale's customer; undefined for who may not look customers up. */
  onPickCustomer?: () => void;
};

/** The screen's shortcuts: F2 goes to the search, F3 chooses the customer, F4 charges and Escape cancels the sale. */
export function usePosShortcuts({
  search,
  dialogOpen,
  canCharge,
  hasLines,
  onCharge,
  onCancel,
  onPickCustomer,
}: Options) {
  useEffect(() => {
    // Each key's action; Escape does nothing with an empty cart.
    const actions: Record<string, (() => void) | undefined> = {
      F2: () => search.current?.focus(),
      F3: onPickCustomer,
      F4: () => {
        if (canCharge) onCharge();
      },
      Escape: hasLines ? onCancel : undefined,
    };
    function onKeyDown(event: KeyboardEvent) {
      const action = actions[event.key];
      if (event.defaultPrevented || dialogOpen || !action) return;
      event.preventDefault();
      action();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });
}
