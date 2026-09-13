import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useRequestRewriteRules } from '../context/rules-context';
import { useBulkDisableRequestRewriteRules } from '../data/rules';

export function RulesBulkDisableDialog() {
  const { t } = useTranslation();
  const { open, setOpen, selectedRules, resetRowSelection } = useRequestRewriteRules();
  const bulkDisableMutation = useBulkDisableRequestRewriteRules();

  const handleConfirm = useCallback(async () => {
    await bulkDisableMutation.mutateAsync(selectedRules.map((rule) => rule.id));
    setOpen(null);
    resetRowSelection?.();
  }, [bulkDisableMutation, resetRowSelection, selectedRules, setOpen]);

  return (
    <AlertDialog open={open === 'bulkDisable'} onOpenChange={(isOpen) => !isOpen && setOpen(null)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('requestRewriteRules.dialogs.bulkDisable.title')}</AlertDialogTitle>
          <AlertDialogDescription>{t('requestRewriteRules.dialogs.bulkDisable.description', { count: selectedRules.length })}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t('common.buttons.cancel')}</AlertDialogCancel>
          <AlertDialogAction onClick={handleConfirm} disabled={bulkDisableMutation.isPending}>
            {t('common.buttons.confirm')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
