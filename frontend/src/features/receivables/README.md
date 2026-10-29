# receivables

El crédito de los clientes y la cartera (`/customers/{id}/credit`, `/customers/{id}/payments`, `/receivables/aging` en
`api/openapi.yaml`). Decisiones de T32 y T32a en `docs/decisiones.md`.

## Páginas

- `CustomerAccountPage` (`/clientes/:id/cuenta`, `receivables:read`): cupo, plazo, lo que debe y lo disponible
  (`CreditSummary`); el formulario de cupo y plazo (`CreditTermsForm`, con `receivables:manage`); las cuentas abiertas
  con su vencimiento (`OpenReceivables`); el abono (`PaymentDialog`, con `receivables:collect`) y el estado de cuenta
  paginado (`Statement`, `?pagina=`).
- `AgingPage` (`/cartera`, `receivables:read`): lo que debe cada cliente por edad (por vencer, 1–30, 31–60, 61–90 y
  más de 90 días) y los totales de todos; cada cliente enlaza a su cuenta.

## API pública (`index.ts`)

- `useCustomerCredit`: el crédito del cliente de la venta, con el que el POS ofrece «Crédito».
- `receivablesKeys`: una venta confirmada refresca el crédito de su cliente y la cartera.

## Reglas

- Un abono nunca pasa de lo que debe el cliente y no se paga con crédito (`model/payment`); en efectivo entra a la
  caja abierta de quien lo recibe (lo decide la API).
- Las fechas de vencimiento son días en hora de Colombia: `dueState` las compara con `todayInBogota`.
- Sin cupo (0) no se compra a crédito; un cupo menor que lo que debe solo impide comprar más.
