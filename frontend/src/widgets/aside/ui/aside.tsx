import { useEffect, useState } from 'react';
import { cn } from 'cn';
import { useSidebarState } from '@/shared/providers';
import { Button, ChevronLeftIcon, ChevronRightIcon, TooltipProvider } from '@/shared/ui';
import { AsideNav } from './aside-nav';

// Between the phone breakpoint (md) and this width the sidebar shows icons only.
const COMPACT_SCREEN_QUERY = '(max-width: 1020px)';

function useIsCompactScreen() {
  const [isCompact, setIsCompact] = useState(() => window.matchMedia(COMPACT_SCREEN_QUERY).matches);

  useEffect(() => {
    const media = window.matchMedia(COMPACT_SCREEN_QUERY);
    const onChange = () => setIsCompact(media.matches);
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, []);

  return isCompact;
}

/** The sidebar on tablets and desktops. Phones get MobileMenu instead. */
export const Aside = () => {
  const { collapsed: preferCollapsed, setCollapsed } = useSidebarState();
  const isCompactScreen = useIsCompactScreen();
  // Narrow screens are always compact; that is not the user's preference.
  const collapsed = preferCollapsed || isCompactScreen;

  return (
    <div className="group/aside relative hidden shrink-0 md:block">
      <TooltipProvider delay={0}>
        <aside
          className={cn(
            'flex h-full flex-col border-r border-sidebar-border bg-sidebar px-2 py-6 transition-all duration-300',
            collapsed ? 'w-16' : 'w-56',
          )}
        >
          <AsideNav collapsed={collapsed} />
        </aside>
      </TooltipProvider>
      {!isCompactScreen && (
        <Button
          variant="outline"
          size="icon-xs"
          onClick={() => setCollapsed(!collapsed)}
          aria-label={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
          aria-expanded={!collapsed}
          className="absolute top-1/2 -right-3 z-10 -translate-y-1/2 rounded-full border-sidebar-border bg-sidebar text-muted-foreground opacity-0 transition-opacity duration-200 group-hover/aside:opacity-100 focus-visible:opacity-100"
        >
          {collapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
        </Button>
      )}
    </div>
  );
};
