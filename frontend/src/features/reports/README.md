# reports

Los reportes del negocio (T42) para el dueño y el administrador (`reports:read`), sobre `GET /api/v1/reports/*`.

- **Pantallas:** `/reportes` (`ReportsLayout`, que abre las ventas) con una sección por reporte: ventas (`/ventas`, por día, vendedor, medio de pago, categoría o sede), productos vendidos (`/productos`, por producto con el detalle por variante y filtros por categoría y atributo), utilidad por venta (`/utilidad`, solo para quien ve costos), sin movimiento (`/sin-movimiento`), cierres de caja (`/cierres`), control del equipo (`/equipo`) y ajustes de inventario (`/ajustes`).
- **Filtros en la URL:** el periodo y la sede (`?desde=&hasta=&sede=`, por defecto el mes en curso) los comparten todos los reportes: los enlaces de las secciones los conservan (`model/sections.ts`). Cada reporte agrega los suyos (`agrupar`, `orden`, `pagina`...).
- **Costos:** los ve quien edita el catálogo (`model/costs.ts`, el mismo permiso que la API). La API no manda costos, utilidad ni márgenes a los demás.
- **Modelo:** `model/period.ts` (periodo por defecto y lo que la API rechazaría), `model/teamTrend.ts` (la tendencia por usuario y lo que se sale de lo normal: más del doble de su promedio), `model/rowLabels.ts` y `model/attributeChoices.ts`.
- **Tablero del inicio (T43):** `TodaySales` (ventas de hoy contra el mismo día de la semana pasada hasta la misma hora, con `until`) y `TeamAlerts` (lo de esta semana que pasa del doble de lo habitual de cada persona en 8 semanas, con la misma regla de `teamTrend`). Modelo en `model/dashboard/`; los días, en `hooks/useDashboardDays` (avanzan cada 5 minutos).
- **API pública:** `ExportButtons` (Excel o CSV de lo que está en pantalla), que también usa la cartera, y las tarjetas `TodaySales` y `TeamAlerts` del inicio.
