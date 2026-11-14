// The color theme of this device: light or dark once chosen, the OS one until
// then. It is a setting of the person at this screen, not of the business, so
// it lives in localStorage. index.html reads the same key before the first paint.

export type Choice = 'light' | 'dark';
export type Theme = Choice | 'system';

const KEY = 'cuadra.theme';
const listeners = new Set<() => void>();

function isChoice(value: string | null): value is Choice {
  return value === 'light' || value === 'dark';
}

export function loadTheme(): Theme {
  try {
    const saved = localStorage.getItem(KEY);
    return isChoice(saved) ? saved : 'system';
  } catch {
    return 'system';
  }
}

export function saveTheme(theme: Choice) {
  try {
    localStorage.setItem(KEY, theme);
  } catch {
    // It applies now; the next visit follows the system again.
  }
  document.documentElement.dataset.theme = theme;
  listeners.forEach((listener) => {
    listener();
  });
}

export function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/**
 * What the screen shows: the choice, or the OS preference until there is one
 * (without data-theme, styles/themes/ follows prefers-color-scheme).
 */
export function shownTheme(theme: Theme): Choice {
  if (theme !== 'system') return theme;
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}
