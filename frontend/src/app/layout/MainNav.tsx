import { NavLink } from 'react-router';
import { visibleItems, type Access } from '@/features/auth';
import { navItems } from './navItems';
import styles from './MainNav.module.scss';
import { paths } from '../routes/paths';

type Props = {
  access: Access;
  /** Called when an item is chosen, e.g. to close the menu panel on a phone. */
  onNavigate?: () => void;
};

export function MainNav({ access, onNavigate }: Props) {
  const items = visibleItems(navItems, access);
  return (
    <nav aria-label="Menú principal" className={styles.root}>
      <ul className={styles.list}>
        {items.map(({ to, label, icon: Icon }) => (
          <li key={to}>
            {/* NavLink marks the current page with aria-current="page". */}
            <NavLink to={to} end={to === paths.home()} className={styles.link} onClick={onNavigate}>
              <Icon className={styles.icon} />
              <span className={styles.label}>{label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
