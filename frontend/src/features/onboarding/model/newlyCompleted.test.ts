import { aChecklist, clothingSteps, hardwareSteps } from '../testing/checklists';
import { newlyCompleted } from './newlyCompleted';

describe('newlyCompleted', () => {
  it.each([
    { name: 'ferretería', ids: hardwareSteps, done: ['products', 'alternate_units'] as const },
    { name: 'almacén de ropa', ids: clothingSteps, done: ['products', 'variant_products'] as const },
  ])('$name: da los pasos completos que aún no se reportaron', ({ ids, done }) => {
    const checklist = aChecklist(ids, [...done]);

    expect(newlyCompleted(checklist.steps, [])).toEqual(done);
    expect(newlyCompleted(checklist.steps, ['products'])).toEqual(done.slice(1));
  });

  it('un paso pendiente no se reporta aunque ya se hubiera reportado antes', () => {
    const checklist = aChecklist(hardwareSteps, []);

    expect(newlyCompleted(checklist.steps, ['first_sale'])).toEqual([]);
  });
});
