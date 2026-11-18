import { roleLabels } from '@/features/auth';
import { formatMoney, formatQuantity } from '@/shared/format';
import type { AuditEvent } from '../api/types';

type Details = Record<string, string>;
type Describe = (d: Details) => string;

type Role = keyof typeof roleLabels;
const isRole = (value: string): value is Role => value in roleLabels;
const role = (value = '') => (isRole(value) ? roleLabels[value] : value);
const money = (value: string | undefined) => formatMoney(value ?? '0');

const settingLabels: Record<string, string> = {
  address: 'Dirección',
  analytics_consent: 'Analítica de uso',
  city: 'Ciudad',
  default_tax_rate: 'IVA por defecto',
  municipality_code: 'Municipio',
  phone: 'Teléfono',
};

function cashDifference(d: Details): string {
  const short = (d.difference ?? '').startsWith('-');
  const amount = money((d.difference ?? '0').replace('-', ''));
  return `${short ? 'Faltaron' : 'Sobraron'} ${amount} (contó ${money(d.counted)} de ${money(d.expected)})`;
}

const describers: Partial<Record<AuditEvent['action'], Describe>> = {
  'sale.voided': (d) => `Venta de ${money(d.total)}`,
  'purchase.voided': (d) => `Compra de ${money(d.total)}`,
  'sale.discount_applied': (d) => `${money(d.discount)} de descuento sobre ${money(d.gross)}`,
  'variant.price_changed': (d) => `De ${money(d.old_price)} a ${money(d.new_price)}`,
  'stock.adjusted': (d) =>
    `${formatQuantity(d.quantity ?? '0', { signed: true })} en ${d.source === 'stock_count' ? 'una toma física' : 'un ajuste manual'}`,
  'cash.closed_with_difference': cashDifference,
  'credit.terms_changed': (d) =>
    `Cupo de ${money(d.old_limit)} a ${money(d.new_limit)}; plazo de ${d.old_term_days ?? ''} a ${d.new_term_days ?? ''} días`,
  'module.enabled': (d) => d.module ?? '',
  'module.disabled': (d) => d.module ?? '',
  'tenant.settings_changed': (d) =>
    `${settingLabels[d.setting ?? ''] ?? d.setting ?? ''}: de ${d.old ?? ''} a ${d.new ?? ''}`,
  'user.invited': (d) => `Como ${role(d.role)}`,
  'user.role_changed': (d) => `De ${role(d.old_role)} a ${role(d.new_role)}`,
};

/** The figure an event adds, in a sentence: "De $ 4.500 a $ 4.800". Empty when the action says it all. */
export function describeEvent(event: AuditEvent): string {
  return describers[event.action]?.(event.details) ?? '';
}
