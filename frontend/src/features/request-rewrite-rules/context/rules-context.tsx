import React, { createContext, useCallback, useContext, useMemo, useState } from 'react';
import { RequestRewriteRule } from '../data/schema';

type DialogType = 'create' | 'edit' | 'delete' | 'bulkEnable' | 'bulkDisable' | 'bulkDelete' | null;

interface RulesContextType {
  open: DialogType;
  setOpen: (open: DialogType) => void;
  currentRow: RequestRewriteRule | null;
  setCurrentRow: (row: RequestRewriteRule | null) => void;
  selectedRules: RequestRewriteRule[];
  setSelectedRules: (rules: RequestRewriteRule[]) => void;
  resetRowSelection: (() => void) | null;
  setResetRowSelection: (fn: (() => void) | null) => void;
}

const RulesContext = createContext<RulesContextType | undefined>(undefined);

export function RequestRewriteRulesProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState<DialogType>(null);
  const [currentRow, setCurrentRow] = useState<RequestRewriteRule | null>(null);
  const [selectedRules, setSelectedRules] = useState<RequestRewriteRule[]>([]);
  const [resetRowSelection, setResetRowSelection] = useState<(() => void) | null>(null);

  const handleSetOpen = useCallback((nextOpen: DialogType) => {
    setOpen(nextOpen);
    if (nextOpen !== 'edit' && nextOpen !== 'delete') {
      setCurrentRow(null);
    }
  }, []);

  const value = useMemo(
    () => ({
      open,
      setOpen: handleSetOpen,
      currentRow,
      setCurrentRow,
      selectedRules,
      setSelectedRules,
      resetRowSelection,
      setResetRowSelection,
    }),
    [open, handleSetOpen, currentRow, selectedRules, resetRowSelection]
  );

  return <RulesContext.Provider value={value}>{children}</RulesContext.Provider>;
}

export function useRequestRewriteRules() {
  const context = useContext(RulesContext);
  if (!context) {
    throw new Error('useRequestRewriteRules must be used within RequestRewriteRulesProvider');
  }

  return context;
}

export default RequestRewriteRulesProvider;
