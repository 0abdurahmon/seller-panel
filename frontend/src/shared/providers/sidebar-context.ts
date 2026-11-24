import { createContext, useContext } from 'react';

export type SidebarState = {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
};

export const SidebarContext = createContext<SidebarState | null>(null);

export function useSidebarState(): SidebarState {
  const context = useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebarState must be used inside SidebarProvider');
  }
  return context;
}
