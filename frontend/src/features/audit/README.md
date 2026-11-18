# audit

La pantalla de auditoría del dueño (T41), sobre el registro de T38 (`GET /api/v1/audit-events`, permiso `audit:read`).

- **Pantalla:** `/auditoria` (`AuditPage`): qué pasó, quién (nombre y rol que tenía), cuándo y el dato que importa ("De $ 4.500 a $ 4.800"), con enlace a la venta, la compra, la caja o la cuenta del cliente cuando la hay.
- **Filtros en la URL:** acción, usuario y días en Colombia (`?accion=&usuario=&desde=&hasta=&pagina=`).
- **Modelo:** `model/actions.ts` nombra cada acción del catálogo (`docs/auditoria.md`, versión 2) y `model/describeEvent.ts` arma la frase de sus `details`. Una acción nueva del catálogo se agrega en los dos (tsc falla si falta en `actionLabels`).
- Los nombres de los usuarios salen de `useUsers` de `@/features/users`; un usuario que ya no existe se muestra como «Usuario borrado».
