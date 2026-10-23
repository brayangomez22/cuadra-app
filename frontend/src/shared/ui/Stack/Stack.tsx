import clsx from 'clsx';
import type { ElementType, HTMLAttributes } from 'react';
import type { Gap } from './layout';
import styles from './Stack.module.scss';

type Props = HTMLAttributes<HTMLElement> & {
  /** The element to render: section, ul, form... A div by default. */
  as?: ElementType;
  gap?: Gap;
  /** Cross-axis alignment of the children; they stretch by default. */
  align?: 'start' | 'center' | 'end' | 'stretch';
};

/** Children one below the other, separated by a step of the spacing scale. */
export function Stack({ as: Component = 'div', gap = '4', align = 'stretch', className, ...rest }: Props) {
  return <Component className={clsx(styles.root, className)} data-gap={gap} data-align={align} {...rest} />;
}
