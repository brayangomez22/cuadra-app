import { useState } from 'react';

/** Copies a text to the clipboard and says so; where the browser blocks it, the field stays to copy by hand. */
export function useCopy() {
  const [copied, setCopied] = useState(false);
  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }
  return { copied, copy };
}
