# customers

Los clientes de la empresa (`/customers`, `/customer-segments` en `api/openapi.yaml`): personas naturales o jurídicas
con su documento, contacto, segmento y responsabilidades fiscales para la factura electrónica. Decisión de T31 y T31a
en `docs/decisiones.md`.

## Páginas

- `CustomerListPage` (`/clientes`, `customers:read`): búsqueda por nombre o documento, filtro por segmento y estado,
  paginada; la búsqueda y los filtros viven en la URL.
- `CustomerFormPage` (`/clientes/nuevo`, `/clientes/:id/editar`, `customers:write`): el formulario completo, con los
  errores de la API en su campo (`invalid_nit_check_digit`, `customer_document_taken`...).

## API pública (`index.ts`)

- `CustomerPicker`: el cliente de la venta en el POS. «Consumidor final» por defecto, búsqueda rápida y creación al
  vuelo (con `customers:write`) en un diálogo con los campos mínimos.
- `useCustomerSegments` y `customersKeys`: los segmentos, que `settings` crea, renombra y borra.

## Reglas

- Una persona jurídica solo se identifica con NIT; el dígito de verificación lo revisa la API.
- Un cliente no se borra: se desactiva. Un segmento con clientes no se borra.
