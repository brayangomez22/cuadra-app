import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { toDecimalInput } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useDeleteReorderRule, useSetReorderRule } from '../../../api/reorder';
import type { StockLevel } from '../../../api/types';
import { reorderErrorFields, reorderRuleSchema } from '../../../model/reorder/reorderRuleSchema';

type Options = { level: StockLevel; name: string; onDone: () => void };

// The form starts with the level's rule, as the person would type it ("2,5").
function defaults({ reorder }: StockLevel) {
  return {
    min_quantity: reorder ? toDecimalInput(reorder.min_quantity) : '',
    max_quantity: reorder ? toDecimalInput(reorder.max_quantity) : '',
  };
}

/** The minimum and maximum of a variant at a location: save them or remove them. `onDone` closes the dialog. */
export function useReorderRuleForm({ level, name, onDone }: Options) {
  const { toast } = useToast();
  const target = { locationId: level.location_id, variantId: level.variant_id };
  const save = useSetReorderRule(target);
  const remove = useDeleteReorderRule(target);
  const form = useForm({ resolver: zodResolver(reorderRuleSchema), defaultValues: defaults(level) });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, reorderErrorFields);

  const done = (title: string) => () => {
    toast({ tone: 'success', title, description: name });
    onDone();
  };

  const submit = form.handleSubmit((values) => {
    clearFormError();
    save.mutate(values, { onSuccess: done('Mínimo y máximo guardados'), onError: handleError });
  });

  function clear() {
    clearFormError();
    remove.mutate(undefined, { onSuccess: done('Mínimo y máximo quitados'), onError: handleError });
  }

  // Each opening starts from the level as it is now.
  function reset() {
    form.reset(defaults(level));
    clearFormError();
  }

  return { form, formError, submit, clear, reset, isSaving: save.isPending, isClearing: remove.isPending };
}
