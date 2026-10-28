import { useEffect, useRef } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router';
import { useAccess, visibleItems } from '@/features/auth';
import { Page } from '@/shared/ui';
import { sections } from '../../model/sections';
import styles from './SettingsLayout.module.scss';

/** /configuracion: the sections the user may see, and the chosen one. */
export function SettingsLayout() {
  const items = visibleItems(sections, useAccess());
  const nav = useRef<HTMLElement>(null);
  const { pathname } = useLocation();

  // On a phone the sections are a row that scrolls: bring the current one in,
  // without moving the page if it is already visible.
  useEffect(() => {
    nav.current?.querySelector('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [pathname]);

  return (
    <Page
      title="Configuración"
      description="Cómo trabaja su negocio en Cuadra: unidades, campos de producto, segmentos de cliente y módulos."
    >
      <div className={styles.root}>
        <nav ref={nav} aria-label="Secciones de configuración" className={styles.nav}>
          <ul className={styles.list}>
            {items.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} className={styles.link}>
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className={styles.content}>
          <Outlet />
        </div>
      </div>
    </Page>
  );
}
