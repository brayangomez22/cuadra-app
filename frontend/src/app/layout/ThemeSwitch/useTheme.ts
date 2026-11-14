import { useSyncExternalStore } from 'react';
import { loadTheme, saveTheme, subscribe } from './theme';

/**
 * The theme chosen on this device ('system' until there is one). One store
 * for the whole page: the side menu and the menu panel on a phone agree.
 */
export function useTheme() {
  const theme = useSyncExternalStore(subscribe, loadTheme);
  return { theme, setTheme: saveTheme };
}
