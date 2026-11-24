import { useCallback, useState, type ReactNode } from 'react';
import { SidebarContext } from './sidebar-context';

const STORAGE_KEY = 'sidebar_collapsed';

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

/** Remembers whether the desktop sidebar is compact, across reloads. */
export function SidebarProvider({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsedState] = useState(readCollapsed);

  const setCollapsed = useCallback((next: boolean) => {
    setCollapsedState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next ? '1' : '0');
    } catch {
      // Storage can be unavailable (private mode); the state still works for this session.
    }
  }, []);

  return (
    <SidebarContext.Provider value={{ collapsed, setCollapsed }}>
      {children}
    </SidebarContext.Provider>
  );
}
