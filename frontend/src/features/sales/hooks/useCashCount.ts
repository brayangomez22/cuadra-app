import { useState } from 'react';
import { useCloseCashSession } from '../api/cashSessions';
import type { CashSession } from '../api/types';
import { countedTotal, readCounts } from '../model/cash/denominations';

/**
 * The count of the drawer as it is typed, and its closing: "Cerrar caja"
 * checks the count and asks to confirm; confirming closes the session.
 */
export function useCashCount(session: CashSession, onClosed: (id: string) => void) {
  const close = useCloseCashSession(session.id);
  const [texts, setTexts] = useState<Record<string, string>>({});
  const [notes, setNotes] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [confirming, setConfirming] = useState(false);

  const { counts, invalid } = readCounts(texts);
  const total = countedTotal(counts);

  function setCount(denomination: string, text: string) {
    setTexts((t) => ({ ...t, [denomination]: text }));
  }

  function submit() {
    setSubmitted(true);
    if (invalid.size > 0) return;
    close.reset();
    setConfirming(true);
  }

  // mutateAsync, not mutate's callbacks: the closed session can replace the
  // form before they run (another user's drawer goes to its report). The
  // error stays in close.error for the dialog.
  async function confirm() {
    const trimmed = notes.trim();
    try {
      const closed = await close.mutateAsync({ counts, ...(trimmed && { notes: trimmed }) });
      onClosed(closed.id);
    } catch {
      // close.isError shows it.
    }
  }

  return {
    texts,
    setCount,
    notes,
    setNotes,
    submitted,
    invalid,
    total,
    submit,
    confirming,
    setConfirming,
    confirm,
    close,
  };
}
