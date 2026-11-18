import type { AuditAction } from '../api/types';

/** What each action of the audit catalog (docs/auditoria.md) means, in a few words. */
export const actionLabels: Record<AuditAction, string> = {
  'sale.voided': 'Anuló una venta',
  'sale.discount_applied': 'Dio un descuento',
  'variant.price_changed': 'Cambió un precio',
  'stock.adjusted': 'Ajustó el inventario',
  'cash.closed_with_difference': 'Cerró caja con diferencia',
  'purchase.voided': 'Anuló una compra',
  'credit.terms_changed': 'Cambió el cupo de un cliente',
  'module.enabled': 'Activó un módulo',
  'module.disabled': 'Desactivó un módulo',
  'tenant.settings_changed': 'Cambió un dato de la empresa',
  'user.invited': 'Invitó a un usuario',
  'user.role_changed': 'Cambió el rol de un usuario',
  'user.deactivated': 'Desactivó un usuario',
  'user.activated': 'Reactivó un usuario',
  'user.password_reset': 'Generó un enlace de contraseña',
};

/** The actions as filter options, in the order of the list above. */
export const actionOptions = (Object.keys(actionLabels) as AuditAction[]).map((action) => ({
  value: action,
  label: actionLabels[action],
}));
