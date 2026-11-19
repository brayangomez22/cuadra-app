import { useEffect, useRef, type ReactNode } from 'react';
import { NavLink, useLocation } from 'react-router';
import { Page } from '../Page';
import styles from './SectionLayout.module.scss';

export type Section = { to: string; label: string };

type Props = {
  title: string;
  description: string;
  /** The accessible name of the sections' navigation. */
  label: string;
  sections: Section[];
  children: ReactNode;
};

/**
 * A screen split in sections (settings, reports): their links, above the
 * content on a phone and as a column at the start from lg, and the current one.
 */
export function SectionLayout({ title, description, label, sections, children }: Props) {
  const nav = useRef<HTMLElement>(null);
  const { pathname } = useLocation();

  // On a phone the sections are a row that scrolls: bring the current one in,
  // without moving the page if it is already visible.
  useEffect(() => {
    nav.current?.querySelector('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [pathname]);

  return (
    <Page title={title} description={description}>
      <div className={styles.root}>
        <nav ref={nav} aria-label={label} className={styles.nav}>
          <ul className={styles.list}>
            {sections.map((section) => (
              <li key={section.to}>
                <NavLink to={section.to} className={styles.link}>
                  {section.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className={styles.content}>{children}</div>
      </div>
    </Page>
  );
}
