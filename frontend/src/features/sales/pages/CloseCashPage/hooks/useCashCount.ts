import { useState } from 'react';
import { useCloseCashSession } from '../../../api/cashSessions';
import type { CashSession } from '../../../api/types';
import { countedTotal, readCounts } from '../../../model/denominations';

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

  function confirm() {
    const trimmed = notes.trim();
    close.mutate({ counts, ...(trimmed && { notes: trimmed }) }, { onSuccess: (closed) => onClosed(closed.id) });
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
