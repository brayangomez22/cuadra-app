# Registro de auditoría: catálogo de acciones

Quién hizo qué, sobre qué, dónde y cuándo, con el dato que los reportes suman. Es la fuente de las métricas de «control del equipo» (anulaciones, descuentos, cambios de precio, ajustes y diferencias de caja por usuario) que muestran T42 y T43, y de la pantalla de auditoría de T41.

Decisión: `docs/decisiones.md`, 2026-10-18 (diseño) y 2026-11-15 (T38).

## Reglas

- **Catálogo cerrado y versionado.** Una acción que no está aquí se rechaza (`audit.ErrUnknownAction`). El dominio (`internal/modules/audit/domain/action.go`) es la fuente de verdad y la constante `CatalogVersion` sube con cada cambio. Una acción nueva o una clave nueva de `details` se agrega aquí y en el dominio en la misma tarea.
- **Nombres neutros**, sin términos de un tipo de negocio: `sale`, `variant`, `stock`, `cash`.
- **`details` exacto.** Cada acción declara sus claves: todas son obligatorias y no se acepta ninguna otra. Así no se cuela un dato personal de un comprador ni texto libre. Los valores son strings de hasta 200 caracteres: los montos con dos decimales (`"12500.00"`) y las cantidades en la unidad base (`"-2.5"`), nunca `float`. Un valor vacío es un dato que se borró.
- **Misma transacción.** El evento se escribe en la transacción de la operación: si la operación falla, no queda evento; si el registro falla, la operación se deshace. Los módulos escriben por su puerto `AuditLog` (`app/audit.go`), que satisface `internal/server/ports/audit.go`.
- **Inmutable.** `app_user` solo tiene `SELECT` e `INSERT` sobre `audit_events`. La purga de la demo los borra como `demo_purge`.
- **Quién.** El usuario y su rol salen del token. Una operación sin token (un trabajo en segundo plano, como una importación) toma el rol actual del usuario que la originó.
- **Consulta.** `GET /api/v1/audit-events` (permiso `audit:read`, solo el dueño), filtrable por acción, usuario, entidad y rango de días.

## Versión 1 (T38)

| Acción | Entidad (`entity_type`) | Sede | `details` | La emite |
|---|---|---|---|---|
| `sale.voided` | `sale` | sí | `total`: total de la venta anulada | `sales.VoidSale` |
| `sale.discount_applied` | `sale` | sí | `discount`: suma de los descuentos de sus líneas; `gross`: subtotal antes del descuento. Un evento por venta con descuento | `sales.ConfirmSale` |
| `variant.price_changed` | `variant` | no | `old_price`, `new_price` (sin IVA). Solo si el precio cambia; una variante nueva no es un cambio | `catalog.UpdateProduct`, `catalog.UpdateVariant` y la importación que actualiza productos |
| `stock.adjusted` | `variant` | sí | `quantity`: el ajuste con signo, en la unidad base; `source`: `adjustment` (ajuste manual) o `stock_count` (aprobación de un conteo, un evento por línea ajustada). El motivo escrito queda en el kardex | `inventory.AdjustStock`, `inventory.ApproveStockCount` |
| `cash.closed_with_difference` | `cash_session` | sí | `expected`, `counted`, `difference` (`counted − expected`: negativo es faltante). Solo si hay diferencia | `sales.CloseCashSession` |
| `purchase.voided` | `purchase` | sí | `total`: total de la compra anulada | `purchases.VoidPurchase` |
| `module.enabled` / `module.disabled` | `module` (sin `entity_id`) | no | `module`: el ID del módulo. Activar los módulos de la plantilla al registrarse no deja eventos | `PUT /api/v1/modules/{id}` |
| `tenant.settings_changed` | `tenant` | no | `setting` (`address`, `analytics_consent`, `city`, `default_tax_rate`, `municipality_code`, `phone`), `old`, `new`. Un evento por dato que cambia | `identity.UpdateTenant` |

## Versión 2 (T41)

| Acción | Entidad (`entity_type`) | Sede | `details` | La emite |
|---|---|---|---|---|
| `user.invited` | `invitation` | no | `role`: el rol con que se invita. Ni el correo ni el nombre del invitado (datos personales) | `identity.InviteUser` |
| `user.role_changed` | `user` | no | `old_role`, `new_role` | `identity.ChangeUserRole` |
| `user.deactivated` | `user` | no | ninguno | `identity.DeactivateUser` |
| `user.activated` | `user` | no | ninguno | `identity.ActivateUser` |
| `user.password_reset` | `user` | no | ninguno: se generó un enlace para que el usuario elija contraseña (nunca el enlace) | `identity.IssuePasswordReset` |
| `credit.terms_changed` | `customer` | no | `old_limit`, `new_limit` (COP con dos decimales), `old_term_days`, `new_term_days`. Solo si algo cambia | `receivables.SetCreditTerms` |

Aceptar una invitación no se audita aparte: el usuario nuevo ya queda en `user.invited` (quién lo invitó y con qué rol) y en la lista de usuarios.

## Lo que viene
- **T37** (notas crédito): la nota crédito de una venta facturada.
