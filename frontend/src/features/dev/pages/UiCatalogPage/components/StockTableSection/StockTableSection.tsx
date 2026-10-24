import { useState } from 'react';
import { Badge, IconAlert, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import { STOCK } from '../../../../model/samples';
import { Section } from '../Section/Section';

function StockBadge({ state }: { state: (typeof STOCK)[number]['state'] }) {
  if (state === 'ok') return <Badge tone="success">Disponible</Badge>;
  return (
    <Badge tone={state === 'low' ? 'warning' : 'danger'} icon={<IconAlert />}>
      {state === 'low' ? 'Stock bajo' : 'Agotado'}
    </Badge>
  );
}

/** A Table with numeric columns, badges and a selected row (click to move it). */
export function StockTableSection() {
  const [selectedSku, setSelectedSku] = useState<string>('CAB-12');
  return (
    <Section title="Table">
      <Table caption="Existencias en la sede Principal" showCaption>
        <TableHead>
          <TableRow>
            <TableHeaderCell>SKU</TableHeaderCell>
            <TableHeaderCell>Producto</TableHeaderCell>
            <TableHeaderCell>Unidad</TableHeaderCell>
            <TableHeaderCell numeric>Cantidad</TableHeaderCell>
            <TableHeaderCell numeric>Precio</TableHeaderCell>
            <TableHeaderCell>Estado</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {STOCK.map((item) => (
            <TableRow key={item.sku} selected={item.sku === selectedSku} onClick={() => setSelectedSku(item.sku)}>
              <TableCell>{item.sku}</TableCell>
              <TableCell>{item.name}</TableCell>
              <TableCell>{item.unit}</TableCell>
              <TableCell numeric>{item.qty}</TableCell>
              <TableCell numeric>{item.price}</TableCell>
              <TableCell>
                <StockBadge state={item.state} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  );
}
