import type { ReactNode, SVGProps } from 'react';

// Inline icons, no library (decision in docs/decisiones.md). 24×24 grid, 2px
// stroke in currentColor, sized at 1em so they follow the text around them.
// Always decorative: the accessible name comes from the button or the text.
type IconProps = Omit<SVGProps<SVGSVGElement>, 'children'>;

export function createIcon(paths: ReactNode) {
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
