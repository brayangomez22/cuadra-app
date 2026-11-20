import { paths } from '@/app/routes/paths';
import type { StepId } from '../api/types';

/** What a step says and where its action goes; a step without action is resolved elsewhere. */
export type StepContent = { title: string; description: string; action?: { label: string; to: string } };

/** The text of each step of the catalog. Neutral: every kind of business reads the same words. */
export const stepContent: Record<StepId, StepContent> = {
  business_profile: {
    title: 'Complete los datos del negocio',
    description: 'La dirección, la ciudad y el teléfono salen en la tirilla y en el cierre de caja.',
    action: { label: 'Completar datos', to: paths.settings.general() },
  },
  // Its screen arrives with the electronic invoicing screens (T36c).
  invoicing: {
    title: 'Registre la resolución de facturación',
    description: 'Sin una resolución de numeración vigente de la DIAN no se emiten documentos electrónicos.',
  },
  products: {
    title: 'Cargue sus productos',
    description: 'Impórtelos desde Excel o créelos uno por uno.',
    action: { label: 'Importar desde Excel', to: paths.imports.new() },
  },
  alternate_units: {
    title: 'Venda en otras unidades',
    description: 'Por ejemplo, por caja y por unidad. Se configura en cada producto.',
    action: { label: 'Ver productos', to: paths.products.list() },
  },
  variant_products: {
    title: 'Cree un producto con variantes',
    description: 'Un producto que se vende en varias versiones, cada una con su código, su precio y su stock.',
    action: { label: 'Nuevo producto', to: paths.products.new() },
  },
  initial_stock: {
    title: 'Cargue el inventario inicial',
    description: 'Registre lo que tiene hoy en cada sede o bodega.',
    action: { label: 'Ir al inventario', to: paths.inventory.stock() },
  },
  reorder_rules: {
    title: 'Defina los mínimos de stock',
    description: 'Cuadra le avisa qué pedir cuando algo baja del mínimo.',
    action: { label: 'Ver qué reponer', to: paths.inventory.reorder() },
  },
  customer_credit: {
    title: 'Dé cupo de crédito a sus clientes',
    description: 'Registre a quién le vende a crédito y hasta cuánto.',
    action: { label: 'Ver clientes', to: paths.customers.list() },
  },
  invite_team: {
    title: 'Invite a su equipo',
    description: 'Cada persona entra con su propio usuario y su rol.',
    action: { label: 'Invitar', to: paths.users.list() },
  },
  first_sale: {
    title: 'Abra la caja y haga la primera venta',
    description: 'La caja lleva lo que entra y sale del cajón en cada turno.',
    action: { label: 'Ir a la caja', to: paths.sales.cash() },
  },
};
