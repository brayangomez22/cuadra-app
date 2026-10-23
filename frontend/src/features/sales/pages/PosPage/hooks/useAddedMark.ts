import { useEffect, useState } from 'react';

// How long the line just added stays marked: enough to notice it, short
// enough that a fast reader does not pile up marks.
const ADDED_MARK_MS = 1500;

/** The line just added (its lineKey); seq changes on every add. */
export type AddedMark = { key: string; seq: number };

/** The mark of the line just added (or added again), which goes off by itself. */
export function useAddedMark() {
  const [added, setAdded] = useState<AddedMark>();

  useEffect(() => {
    if (!added) return;
    const timer = setTimeout(() => setAdded(undefined), ADDED_MARK_MS);
    return () => clearTimeout(timer);
  }, [added]);

  function mark(key: string) {
    const next = { key, seq: (added?.seq ?? 0) + 1 };
    if (added?.key === key) {
      // Scanned again while marked: take the mark off for a frame so it starts over.
      setAdded(undefined);
      requestAnimationFrame(() => setAdded(next));
    } else {
      setAdded(next);
    }
  }

  return { added, mark };
}
