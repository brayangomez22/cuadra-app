import { useState } from 'react';
import { QueryState } from '@/shared/query';
import { Pagination, Spinner, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import { ERRORS_PAGE_SIZE, useImportErrors } from '../../api/imports';
import styles from './ImportErrors.module.scss';

/** Every error of an import's rows, by row, a page at a time (the API keeps the first 1.000). */
export function ImportErrors({ importId }: { importId: string }) {
  const [offset, setOffset] = useState(0);
  const errors = useImportErrors(importId, offset);
  return (
    <QueryState
      queries={[errors]}
      errorTitle="No pudimos cargar los errores del archivo"
      loading={<Spinner label="Cargando errores…" />}
    >
      {() =>
        errors.data && (
          <div className={styles.root}>
            <Table caption="Errores del archivo">
              <TableHead>
                <TableRow>
                  <TableHeaderCell numeric>Fila</TableHeaderCell>
                  <TableHeaderCell>Columna</TableHeaderCell>
                  <TableHeaderCell>Problema</TableHeaderCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {errors.data.items.map((error) => (
                  <TableRow key={`${String(error.row)}-${error.column}-${error.code}`}>
                    <TableCell numeric>{error.row}</TableCell>
                    <TableCell>{error.column}</TableCell>
                    <TableCell>{error.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              label="Páginas de errores"
              total={errors.data.total}
              limit={ERRORS_PAGE_SIZE}
              offset={offset}
              onOffsetChange={setOffset}
            />
          </div>
        )
      }
    </QueryState>
  );
}
