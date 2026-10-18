import { Navigate, NavLink, Outlet } from 'react-router';
import { useAccess, visibleItems } from '@/features/auth';
import { Page } from '@/shared/ui';
import { sections } from './sections';
import styles from './SettingsLayout.module.scss';

/** /configuracion: the sections the user may see, and the chosen one. */
export function SettingsLayout() {
  const items = visibleItems(sections, useAccess());
  return (
    <Page title="Configuración" description="Cómo trabaja su negocio en Cuadra: unidades, campos de producto y módulos.">
      <div className={styles.root}>
        <nav aria-label="Secciones de configuración" className={styles.nav}>
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

/** /configuracion opens the first section the user may see. */
export function SettingsHome() {
  const [first] = visibleItems(sections, useAccess());
  return first ? <Navigate to={first.to} replace /> : null;
}
