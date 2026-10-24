import { isZero } from '@/shared/format';

/** What goes out of the drawer, with its sign; zero stays zero. */
export function negated(amount: string): string {
  return isZero(amount) ? amount : `-${amount}`;
}
