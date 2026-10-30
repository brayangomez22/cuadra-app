import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Location, StockMovement } from '../../../../api/types';
import { MovementRow } from '../MovementRow/MovementRow';

type Props = {
  caption: string;
  movements: StockMovement[];
  locations: Location[];
  /** Whether the kardex shows every location: each balance then says whose it is. */
  allLocations: boolean;
  canSeeCost: boolean;
};

/**
 * The movements with their balance; the unit cost only for those who see
 * costs, and the detail only when a movement of the page has one.
 */
export function MovementsTable({ caption, movements, locations, allLocations, canSeeCost }: Props) {
  const locationName = new Map(locations.map((l) => [l.id, l.name]));
  const withDetail = movements.some((m) => m.reason);
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell wide>Fecha</TableHeaderCell>
          <TableHeaderCell>Movimiento</TableHeaderCell>
          <TableHeaderCell numeric>Cantidad</TableHeaderCell>
          <TableHeaderCell numeric>{allLocations ? 'Saldo en su sede' : 'Saldo en la sede'}</TableHeaderCell>
          {canSeeCost && (
            <TableHeaderCell numeric wide>
              Costo unitario
            </TableHeaderCell>
          )}
          {withDetail && <TableHeaderCell wide>Detalle</TableHeaderCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {movements.map((movement) => (
          <MovementRow
            key={movement.id}
            movement={movement}
            locationName={allLocations ? locationName.get(movement.location_id) : undefined}
            canSeeCost={canSeeCost}
            withDetail={withDetail}
          />
        ))}
      </TableBody>
    </Table>
  );
}
