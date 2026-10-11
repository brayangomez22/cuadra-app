import type { ReactNode, SVGProps } from 'react';

// Inline icons, no library (decision in docs/decisiones.md). 24×24 grid, 2px
// stroke in currentColor, sized at 1em so they follow the text around them.
// Always decorative: the accessible name comes from the button or the text.
type IconProps = Omit<SVGProps<SVGSVGElement>, 'children'>;

function createIcon(paths: ReactNode) {
  return function Icon(props: IconProps) {
    return (
      <svg
        width="1em"
        height="1em"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
        focusable="false"
        {...props}
      >
        {paths}
      </svg>
    );
  };
}

export const IconClose = createIcon(<path d="M18 6 6 18M6 6l12 12" />);
export const IconCheck = createIcon(<path d="M20 6 9 17l-5-5" />);
export const IconChevronDown = createIcon(<path d="m6 9 6 6 6-6" />);
export const IconPlus = createIcon(<path d="M12 5v14M5 12h14" />);
export const IconSearch = createIcon(
  <>
    <circle cx="11" cy="11" r="7" />
    <path d="m20 20-3.5-3.5" />
  </>,
);
export const IconAlert = createIcon(
  <>
    <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
    <path d="M12 9v4M12 17h.01" />
  </>,
);
export const IconInfo = createIcon(
  <>
    <circle cx="12" cy="12" r="10" />
    <path d="M12 16v-4M12 8h.01" />
  </>,
);
export const IconSuccess = createIcon(
  <>
    <circle cx="12" cy="12" r="10" />
    <path d="m8 12 3 3 5-6" />
  </>,
);
export const IconBox = createIcon(
  <>
    <path d="M21 8 12 3 3 8v8l9 5 9-5V8Z" />
    <path d="m3 8 9 5 9-5M12 13v8" />
  </>,
);
