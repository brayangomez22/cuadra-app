import clsx from 'clsx';
import type { ElementType, HTMLAttributes } from 'react';
import type { Gap } from '../Stack';
import styles from './Inline.module.scss';

type Props = HTMLAttributes<HTMLElement> & {
  /** The element to render: ul, header, footer... A div by default. */
  as?: ElementType;
  gap?: Gap;
  /** Cross-axis alignment; centered by default. */
  align?: 'start' | 'center' | 'end' | 'baseline' | 'stretch';
  justify?: 'start' | 'center' | 'end' | 'between';
  /** Let the children wrap to a new line on narrow screens. */
  wrap?: boolean;
};

/** Children side by side, separated by a step of the spacing scale. */
export function Inline({
  as: Component = 'div',
  gap = '2',
  align = 'center',
  justify = 'start',
  wrap = false,
  className,
  ...rest
}: Props) {
  return (
    <Component
      className={clsx(styles.root, className)}
      data-gap={gap}
      data-align={align}
      data-justify={justify}
      data-wrap={wrap}
      {...rest}
    />
  );
}
