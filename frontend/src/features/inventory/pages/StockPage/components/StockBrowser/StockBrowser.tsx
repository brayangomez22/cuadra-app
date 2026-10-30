import type { Location } from '../../../../api/types';
import { useStockSearch } from '../../hooks/useStockSearch';
import { LocationStock } from '../LocationStock/LocationStock';
import { StockFilters } from '../StockFilters/StockFilters';

type Props = {
  locations: Location[];
  location: Location;
  onLocationChange: (id: string) => void;
  offset: number;
  onOffsetChange: (offset: number) => void;
  canSeeCost: boolean;
  canReceive: boolean;
};

/** The stock of the chosen location, searched by name, SKU or barcode (?buscar=). */
export function StockBrowser({ locations, location, onLocationChange, ...stock }: Props) {
  const search = useStockSearch();
  return (
    <>
      <StockFilters
        locations={locations}
        location={location}
        onLocationChange={onLocationChange}
        text={search.text}
        onTextChange={search.setText}
      />
      <LocationStock location={location} term={search.term} {...stock} />
    </>
  );
}
