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
import { useBulkDeleteRequestRewriteRules } from '../data/rules';

export function RulesBulkDeleteDialog() {
  const { t } = useTranslation();
  const { open, setOpen, selectedRules, resetRowSelection } = useRequestRewriteRules();
  const bulkDeleteMutation = useBulkDeleteRequestRewriteRules();

  const handleConfirm = useCallback(async () => {
    await bulkDeleteMutation.mutateAsync(selectedRules.map((rule) => rule.id));
    setOpen(null);
    resetRowSelection?.();
  }, [bulkDeleteMutation, resetRowSelection, selectedRules, setOpen]);

  return (
    <AlertDialog open={open === 'bulkDelete'} onOpenChange={(isOpen) => !isOpen && setOpen(null)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('requestRewriteRules.dialogs.bulkDelete.title')}</AlertDialogTitle>
          <AlertDialogDescription>{t('requestRewriteRules.dialogs.bulkDelete.description', { count: selectedRules.length })}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t('common.buttons.cancel')}</AlertDialogCancel>
          <AlertDialogAction onClick={handleConfirm} disabled={bulkDeleteMutation.isPending} className='bg-destructive text-destructive-foreground hover:bg-destructive/90'>
            {t('common.buttons.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
