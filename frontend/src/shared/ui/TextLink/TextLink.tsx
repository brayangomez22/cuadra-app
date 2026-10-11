import clsx from 'clsx';
import { Link, type LinkProps } from 'react-router';
import styles from './TextLink.module.scss';

type Props = LinkProps & {
  /** quiet: a gray underline, for links repeated on every row of a table. */
  tone?: 'accent' | 'quiet';
};

/** A link inside text or a table cell: underlined, never orange text (docs/estilos.md). */
export function TextLink({ tone = 'accent', className, ...rest }: Props) {
  return <Link className={clsx(styles.root, className)} data-tone={tone} {...rest} />;
}
