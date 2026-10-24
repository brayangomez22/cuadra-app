import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Location, StockMovement } from '../../../../api/types';
import { MovementRow } from '../MovementRow/MovementRow';

type Props = { caption: string; movements: StockMovement[]; locations: Location[]; canSeeCost: boolean };

/** The movements with their balance; the unit cost only for those who see costs. */
export function MovementsTable({ caption, movements, locations, canSeeCost }: Props) {
  const locationName = new Map(locations.map((l) => [l.id, l.name]));
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Fecha</TableHeaderCell>
          <TableHeaderCell>Movimiento</TableHeaderCell>
          <TableHeaderCell numeric>Cantidad</TableHeaderCell>
          <TableHeaderCell numeric>Saldo en la sede</TableHeaderCell>
          {canSeeCost && <TableHeaderCell numeric>Costo unitario</TableHeaderCell>}
          <TableHeaderCell>Detalle</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {movements.map((movement) => (
          <MovementRow
            key={movement.id}
            movement={movement}
            locationName={locationName.get(movement.location_id)}
            canSeeCost={canSeeCost}
          />
        ))}
      </TableBody>
    </Table>
  );
}
