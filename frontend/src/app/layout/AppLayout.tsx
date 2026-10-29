import clsx from 'clsx';
import { useState } from 'react';
import { Outlet } from 'react-router';
import { useAuth } from '@/features/auth';
import { Drawer, IconButton, IconMenu } from '@/shared/ui';
import { AccountPanel } from './AccountPanel';
import { DemoBanner } from './DemoBanner';
import { MainNav } from './MainNav';
import styles from './AppLayout.module.scss';

// AppLayout is the frame of every screen behind the login. From lg up the menu
// is a fixed side column; below, a top bar opens it in a side panel. It renders
// inside RequireAuth, so there is always a user.
export function AppLayout() {
  const auth = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  if (auth.status !== 'authenticated') return null;
  const { user, logout } = auth;

  return (
    <div className={styles.root}>
      <a href="#contenido" className={styles.skipLink}>
        Saltar al contenido
      </a>

      <header className={styles.topbar}>
        <Drawer
          open={menuOpen}
          onOpenChange={setMenuOpen}
          title="Menú"
          trigger={<IconButton label="Abrir menú" icon={<IconMenu />} />}
        >
          <MainNav access={user} onNavigate={() => setMenuOpen(false)} />
          <AccountPanel user={user} onLogout={logout} />
        </Drawer>
        <span className={styles.brand}>Cuadra</span>
      </header>

      <aside className={styles.sidebar}>
        <span className={clsx(styles.brand, styles.sidebarBrand)}>Cuadra</span>
        <MainNav access={user} />
        <AccountPanel user={user} onLogout={logout} className={styles.sidebarAccount} />
      </aside>

      <main id="contenido" tabIndex={-1} className={styles.main}>
        {user.demo && <DemoBanner />}
        <Outlet />
      </main>
    </div>
  );
}
