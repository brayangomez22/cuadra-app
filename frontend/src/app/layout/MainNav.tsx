import { NavLink } from 'react-router';
import { navItems } from './navItems';
import styles from './MainNav.module.scss';

type Props = {
  permissions: readonly string[];
  /** Called when an item is chosen, e.g. to close the menu panel on a phone. */
  onNavigate?: () => void;
};

export function MainNav({ permissions, onNavigate }: Props) {
  const items = navItems.filter((item) => !item.permission || permissions.includes(item.permission));
  return (
    <nav aria-label="Menú principal" className={styles.root}>
      <ul className={styles.list}>
        {items.map(({ to, label, icon: Icon }) => (
          <li key={to}>
            {/* NavLink marks the current page with aria-current="page". */}
            <NavLink to={to} end={to === '/'} className={styles.link} onClick={onNavigate}>
              <Icon className={styles.icon} />
              <span className={styles.label}>{label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
