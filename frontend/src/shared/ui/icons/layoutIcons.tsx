import { createIcon } from './createIcon';

// A panel with its side column and an arrow: hide the side menu (arrow in)
// or show it (arrow out).
export const IconSidebarClose = createIcon(
  <>
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M9 3v18M16 15l-3-3 3-3" />
  </>,
);
export const IconSidebarOpen = createIcon(
  <>
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M9 3v18M14 9l3 3-3 3" />
  </>,
);

// The theme the button switches to: a sun for light, a moon for dark.
export const IconSun = createIcon(
  <>
    <circle cx="12" cy="12" r="4" />
    <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
  </>,
);
export const IconMoon = createIcon(<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z" />);
