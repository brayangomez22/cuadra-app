import type { ComponentProps, ReactNode } from 'react';
import { Link, useLocation } from 'react-router';
import { Stack } from '@/shared/ui';
import styles from './AuthPage.module.scss';

type Props = {
  title: string;
  description?: ReactNode;
  children: ReactNode;
  /** The way to the other screen ("¿No tiene cuenta? Regístrese"). */
  footer?: ReactNode;
  /** wide fits a two-column form from md up. */
  size?: 'md' | 'wide';
};

// AuthPage is the frame of the login and sign-up: the brand and one card,
// centered, alone on the page.
export function AuthPage({ title, description, children, footer, size = 'md' }: Props) {
  return (
    <main className={styles.root}>
      <Stack gap="4" className={styles.column} data-size={size}>
        <p className={styles.brand}>Cuadra</p>
        <section className={styles.card} aria-labelledby="auth-title">
          <Stack as="header" gap="2">
            <h1 id="auth-title" className={styles.title}>
              {title}
            </h1>
            {description && <p className={styles.description}>{description}</p>}
          </Stack>
          {children}
        </section>
        {footer && <p className={styles.footer}>{footer}</p>}
      </Stack>
    </main>
  );
}

/** A link between the auth screens, keeping the page asked for before the login. */
export function AuthLink(props: Omit<ComponentProps<typeof Link>, 'state'>) {
  // RequireAuth keeps that page in the location state (redirect.ts).
  const state: unknown = useLocation().state;
  return <Link className={styles.link} state={state} {...props} />;
}
