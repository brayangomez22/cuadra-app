import { useState } from 'react';

const KEY = 'cuadra.nav.collapsed';

function load(): boolean {
  try {
    return localStorage.getItem(KEY) === 'true';
  } catch {
    return false;
  }
}

function save(collapsed: boolean) {
  try {
    localStorage.setItem(KEY, String(collapsed));
  } catch {
    // The menu opens again next time.
  }
}

/** Whether the side menu shows only its icons: a setting of this device, open unless chosen. */
export function useNavCollapsed() {
  const [collapsed, setCollapsed] = useState(load);

  function toggle() {
    save(!collapsed);
    setCollapsed(!collapsed);
  }

  return { collapsed, toggle };
}
