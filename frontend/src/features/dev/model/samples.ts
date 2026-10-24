// Sample data of the UI catalog, from several business types (docs/estilos.md,
// "Neutralidad").

export const THEMES = [
  { value: 'system', label: 'Sistema' },
  { value: 'light', label: 'Claro' },
  { value: 'dark', label: 'Oscuro' },
];

export const UNITS = [
  { value: 'und', label: 'Unidad' },
  { value: 'm', label: 'Metro' },
  { value: 'kg', label: 'Kilogramo' },
  { value: 'par', label: 'Par' },
  { value: 'caja', label: 'Caja', disabled: true },
];

// A long name, a fractional quantity, thousands and a sold-out product.
export const STOCK = [
  { sku: 'ACE-500', name: 'Acetaminofén 500 mg × 10 tabletas', unit: 'caja', qty: '12', price: '$ 4.900', state: 'ok' },
  {
    sku: 'CAB-12',
    name: 'Cable eléctrico 12 AWG THHN rojo, se vende por metro',
    unit: 'm',
    qty: '2,5',
    price: '$ 3.450',
    state: 'low',
  },
  { sku: 'CUA-100', name: 'Cuaderno cuadriculado 100 hojas', unit: 'und', qty: '1.240', price: '$ 3.200', state: 'ok' },
  { sku: 'TEN-38', name: 'Tenis de lona blancos (38)', unit: 'par', qty: '0', price: '$ 89.900', state: 'out' },
] as const;
