import type { Me } from '@/features/auth';
import { IconButton, IconSidebarClose, IconSidebarOpen } from '@/shared/ui';
import { AccountPanel } from '../AccountPanel';
import { Brand } from '../Brand/Brand';
import { MainNav } from '../MainNav';
import { useNavCollapsed } from './useNavCollapsed';
import styles from './Sidebar.module.scss';

type Props = {
  user: Me;
  onLogout: () => Promise<void>;
};

/**
 * Sidebar is the menu column from lg up. It folds to its icons to give the
 * screen more room, and remembers it on this device.
 */
export function Sidebar({ user, onLogout }: Props) {
  const { collapsed, toggle } = useNavCollapsed();

  return (
    <aside className={styles.root} data-collapsed={collapsed || undefined}>
      <div className={styles.header}>
        {!collapsed && <Brand />}
        <IconButton
          label={collapsed ? 'Mostrar menú' : 'Ocultar menú'}
          icon={collapsed ? <IconSidebarOpen /> : <IconSidebarClose />}
          aria-expanded={!collapsed}
          onClick={toggle}
        />
      </div>
      <MainNav access={user} compact={collapsed} />
      <AccountPanel user={user} onLogout={onLogout} compact={collapsed} className={styles.account} />
    </aside>
  );
}
