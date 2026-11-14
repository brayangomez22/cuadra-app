import { useState } from 'react';
import { Outlet } from 'react-router';
import { useAuth } from '@/features/auth';
import { Drawer, IconButton, IconMenu } from '@/shared/ui';
import { AccountPanel } from './AccountPanel';
import { Brand } from './Brand/Brand';
import { MainNav } from './MainNav';
import { Sidebar } from './Sidebar/Sidebar';
import styles from './AppLayout.module.scss';

// AppLayout is the frame of every screen behind the login. From lg up the menu
// is a side column that folds to its icons; below, a top bar opens it in a
// side panel. It renders inside RequireAuth, so there is always a user.
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
        <Brand />
      </header>

      <Sidebar user={user} onLogout={logout} />

      <main id="contenido" tabIndex={-1} className={styles.main}>
        <Outlet />
      </main>
    </div>
  );
}
