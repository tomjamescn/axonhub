import { useCallback, useEffect, useMemo } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { IconPlus, IconTrash } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { useRequestRewriteRules } from '../context/rules-context';
import { useCreateRequestRewriteRule, useUpdateRequestRewriteRule } from '../data/rules';

const valueMapSchema = z.object({
  from: z.string(),
  to: z.string(),
});

const fieldMapSchema = z
  .object({
    path: z.string().min(1, 'Field path is required'),
    values: z.array(valueMapSchema),
  })
  .superRefine((val, ctx) => {
    if (val.values.length === 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['values'],
        message: 'At least one value mapping is required',
      });
    }
  });

const createFormSchema = (t: ReturnType<typeof useTranslation>['t']) =>
  z
    .object({
      name: z.string().min(1, t('requestRewriteRules.validation.nameRequired')),
      description: z.string().optional(),
      modelPattern: z.string().min(1, t('requestRewriteRules.validation.modelPatternRequired')),
      fieldMaps: z.array(fieldMapSchema).min(1, t('requestRewriteRules.validation.fieldMapsRequired')),
    })
    .superRefine((val, ctx) => {
      const seen = new Set<string>();
      val.fieldMaps.forEach((field, index) => {
        if (seen.has(field.path)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['fieldMaps', index, 'path'],
            message: t('requestRewriteRules.validation.duplicateFieldPath'),
          });
        }
        seen.add(field.path);
      });
    });

type FormData = z.infer<ReturnType<typeof createFormSchema>>;

const defaultValues: FormData = {
  name: '',
  description: '',
  modelPattern: '',
  fieldMaps: [{ path: '', values: [{ from: '', to: '' }] }],
};

export function RulesActionDialog() {
  const { t } = useTranslation();
  const { open, setOpen, currentRow, setCurrentRow, resetRowSelection } = useRequestRewriteRules();
  const createMutation = useCreateRequestRewriteRule();
  const updateMutation = useUpdateRequestRewriteRule();

  const isEdit = open === 'edit';
  const isOpen = open === 'create' || open === 'edit';
  const formSchema = useMemo(() => createFormSchema(t), [t]);

  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues,
  });

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    if (isEdit && currentRow) {
      form.reset({
        name: currentRow.name,
        description: currentRow.description || '',
        modelPattern: currentRow.modelPattern,
        fieldMaps:
          currentRow.fieldMaps && currentRow.fieldMaps.length > 0
            ? currentRow.fieldMaps.map((field) => ({
                path: field.path,
                values: field.values.length > 0 ? field.values.map((v) => ({ from: v.from, to: v.to })) : [{ from: '', to: '' }],
              }))
            : [{ path: '', values: [{ from: '', to: '' }] }],
      });
      return;
    }

    form.reset(defaultValues);
  }, [currentRow, form, isEdit, isOpen]);

  const { fields: fieldMapFields, append: appendFieldMap, remove: removeFieldMap } = useFieldArray({
    control: form.control,
    name: 'fieldMaps',
  });

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        setOpen(null);
        setCurrentRow(null);
        form.reset(defaultValues);
      }
    },
    [form, setCurrentRow, setOpen]
  );

  const onSubmit = useCallback(
    async (values: FormData) => {
      const fieldMaps = values.fieldMaps.map((field) => ({
        path: field.path,
        values: field.values.map((v) => ({ from: v.from, to: v.to })),
      }));

      const input = {
        name: values.name,
        description: values.description || '',
        modelPattern: values.modelPattern,
        fieldMaps,
      };

      if (isEdit && currentRow) {
        await updateMutation.mutateAsync({
          id: currentRow.id,
          input,
        });
      } else {
        await createMutation.mutateAsync(input);
      }

      setOpen(null);
      setCurrentRow(null);
      resetRowSelection?.();
      form.reset(defaultValues);
    },
    [createMutation, currentRow, form, isEdit, resetRowSelection, setCurrentRow, setOpen, updateMutation]
  );

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-[720px]'>
        <DialogHeader>
          <DialogTitle>{isEdit ? t('requestRewriteRules.dialogs.edit.title') : t('requestRewriteRules.dialogs.create.title')}</DialogTitle>
          <DialogDescription>
            {isEdit ? t('requestRewriteRules.dialogs.edit.description') : t('requestRewriteRules.dialogs.create.description')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('requestRewriteRules.fields.name')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('common.columns.description')}</FormLabel>
                  <FormControl>
                    <Input {...field} value={field.value ?? ''} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='modelPattern'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('requestRewriteRules.fields.modelPattern')}</FormLabel>
                  <FormControl>
                    <Input {...field} className='font-mono text-xs' placeholder='gpt-4o' />
                  </FormControl>
                  <FormDescription>{t('requestRewriteRules.fields.modelPatternHint')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className='space-y-3'>
              <div className='flex items-center justify-between'>
                <FormLabel>{t('requestRewriteRules.fields.fieldMaps')}</FormLabel>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => appendFieldMap({ path: '', values: [{ from: '', to: '' }] })}
                >
                  <IconPlus className='mr-2 h-4 w-4' />
                  {t('requestRewriteRules.fields.addField')}
                </Button>
              </div>
              <p className='text-muted-foreground text-sm'>{t('requestRewriteRules.fields.fieldMapsHint')}</p>
              {form.formState.errors.fieldMaps?.root?.message && (
                <p className='text-destructive text-sm'>{form.formState.errors.fieldMaps.root.message}</p>
              )}
              {fieldMapFields.map((fieldMap, fieldMapIndex) => (
                <div key={fieldMap.id} className='space-y-3 rounded-lg border p-4'>
                  <div className='flex items-start gap-2'>
                    <div className='flex-1'>
                      <FormField
                        control={form.control}
                        name={`fieldMaps.${fieldMapIndex}.path`}
                        render={({ field: pathField }) => (
                          <FormItem>
                            <FormLabel>{t('requestRewriteRules.fields.fieldPath')}</FormLabel>
                            <FormControl>
                              <Input {...pathField} className='font-mono text-xs' placeholder='reasoning_effort' />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    {fieldMapFields.length > 1 && (
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon'
                        className='mt-8 text-destructive h-8 w-8 hover:bg-red-100 hover:text-red-700'
                        onClick={() => removeFieldMap(fieldMapIndex)}
                      >
                        <IconTrash className='h-4 w-4' />
                      </Button>
                    )}
                  </div>

                  <ValueMapsEditor control={form.control} fieldMapIndex={fieldMapIndex} />
                </div>
              ))}
            </div>

            <DialogFooter>
              <Button type='button' variant='outline' onClick={() => handleOpenChange(false)}>
                {t('common.buttons.cancel')}
              </Button>
              <Button type='submit' disabled={createMutation.isPending || updateMutation.isPending}>
                {isEdit ? t('common.buttons.update') : t('common.buttons.create')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function ValueMapsEditor({ control, fieldMapIndex }: { control: any; fieldMapIndex: number }) {
  const { t } = useTranslation();
  const { fields, append, remove } = useFieldArray({
    control,
    name: `fieldMaps.${fieldMapIndex}.values`,
  });

  return (
    <div className='space-y-2'>
      <div className='flex items-center justify-between'>
        <span className='text-muted-foreground text-sm font-medium'>{t('requestRewriteRules.fields.valueMaps')}</span>
        <Button type='button' variant='ghost' size='sm' className='h-7' onClick={() => append({ from: '', to: '' })}>
          <IconPlus className='mr-1 h-3 w-3' />
          {t('requestRewriteRules.fields.addValue')}
        </Button>
      </div>
      {fields.map((valueMap, valueMapIndex) => (
        <div key={valueMap.id} className='flex items-center gap-2'>
          <FormField
            control={control}
            name={`fieldMaps.${fieldMapIndex}.values.${valueMapIndex}.from`}
            render={({ field }) => (
              <Input
                className='font-mono h-8 flex-1 text-xs'
                placeholder={t('requestRewriteRules.fields.fromPlaceholder')}
                value={field.value ?? ''}
                onChange={field.onChange}
                onBlur={field.onBlur}
                ref={field.ref}
              />
            )}
          />
          <span className='text-muted-foreground text-sm'>→</span>
          <FormField
            control={control}
            name={`fieldMaps.${fieldMapIndex}.values.${valueMapIndex}.to`}
            render={({ field }) => (
              <Input
                className='font-mono h-8 flex-1 text-xs'
                placeholder={t('requestRewriteRules.fields.toPlaceholder')}
                value={field.value ?? ''}
                onChange={field.onChange}
                onBlur={field.onBlur}
                ref={field.ref}
              />
            )}
          />
          {fields.length > 1 && (
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-destructive h-8 w-8 hover:bg-red-100 hover:text-red-700'
              onClick={() => remove(valueMapIndex)}
            >
              <IconTrash className='h-4 w-4' />
            </Button>
          )}
        </div>
      ))}
      <p className='text-muted-foreground text-xs'>{t('requestRewriteRules.fields.valueMapsHint')}</p>
    </div>
  );
}
