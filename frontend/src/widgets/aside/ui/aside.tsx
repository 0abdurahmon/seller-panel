import { useEffect, useState } from 'react';
import { Link, useMatch } from 'react-router';
import { ArrowLeft2, ArrowRight2 } from 'iconsax-reactjs';
import { cn } from 'cn';
import { bottomMenuItems, mainMenuItems, type MenuItem } from '@/shared/config';
import { useSidebarState } from '@/shared/providers';
import { Logo, LogoMark, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/shared/ui';

const SMALL_SCREEN_QUERY = '(max-width: 1020px)';

function useIsSmallScreen() {
  const [isSmall, setIsSmall] = useState(() => window.matchMedia(SMALL_SCREEN_QUERY).matches);

  useEffect(() => {
    const media = window.matchMedia(SMALL_SCREEN_QUERY);
    const onChange = () => setIsSmall(media.matches);
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, []);

  return isSmall;
}

function MenuLink({ item, collapsed }: { item: MenuItem; collapsed: boolean }) {
  // "/" is active only on the home page; other items stay active on nested pages (/products/42).
  const isActive = Boolean(useMatch({ path: item.href, end: item.href === '/' }));
  const ItemIcon = item.icon;

  const element = (
    <Link
      to={item.href}
      aria-current={isActive ? 'page' : undefined}
      className={cn(
        'font-medium text-sm flex items-center justify-start py-3 rounded-xl transition-colors duration-200 overflow-hidden whitespace-nowrap w-full text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/30',
        collapsed ? 'px-3.5' : 'px-3.5 gap-x-3',
        isActive
          ? 'bg-primary text-primary-foreground dark:bg-white/8 dark:text-foreground'
          : 'text-muted-foreground hover:bg-accent hover:text-foreground',
      )}
    >
      <ItemIcon
        variant={isActive ? 'Bold' : 'Linear'}
        className={cn('size-5 shrink-0', isActive && 'dark:text-primary')}
      />
      <span
        className={cn(
          'transition-all duration-300 flex-1',
          collapsed ? 'w-0 opacity-0' : 'w-auto opacity-100',
        )}
      >
        {item.label}
      </span>
    </Link>
  );

  return (
    <li>
      {collapsed ? (
        <Tooltip>
          <TooltipTrigger render={element} />
          <TooltipContent side="right">{item.label}</TooltipContent>
        </Tooltip>
      ) : (
        element
      )}
    </li>
  );
}

export const Aside = () => {
  const { collapsed: preferCollapsed, setCollapsed } = useSidebarState();
  const isSmallScreen = useIsSmallScreen();
  // Narrow screens are always compact; that is not the user's preference.
  const collapsed = preferCollapsed || isSmallScreen;

  return (
    <div className="relative shrink-0 group/aside">
      <TooltipProvider delay={0}>
        <aside
          className={cn(
            'bg-sidebar border-r border-sidebar-border py-6 px-2 h-full flex flex-col transition-all duration-300',
            collapsed ? 'w-16' : 'w-56',
          )}
        >
          <Link
            to="/"
            aria-label="Dona"
            className={cn(
              'mb-6 flex h-8 items-center rounded-xl text-foreground outline-none focus-visible:ring-3 focus-visible:ring-ring/30',
              collapsed ? 'justify-center' : 'px-3.5',
            )}
          >
            {collapsed ? <LogoMark className="size-8 shrink-0" /> : <Logo className="h-8 w-auto" />}
          </Link>
          <nav aria-label="Основное меню" className="flex flex-1 flex-col">
            <ul className="space-y-1">
              {mainMenuItems.map((item) => (
                <MenuLink key={item.href} item={item} collapsed={collapsed} />
              ))}
            </ul>
            <ul className="space-y-1 mt-auto">
              {bottomMenuItems.map((item) => (
                <MenuLink key={item.href} item={item} collapsed={collapsed} />
              ))}
            </ul>
          </nav>
        </aside>
      </TooltipProvider>
      {!isSmallScreen && (
        <button
          type="button"
          onClick={() => setCollapsed(!collapsed)}
          aria-label={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
          aria-expanded={!collapsed}
          className="absolute top-1/2 -translate-y-1/2 -right-3 size-6 rounded-full bg-sidebar border border-sidebar-border flex items-center justify-center text-muted-foreground hover:text-foreground opacity-0 group-hover/aside:opacity-100 focus-visible:opacity-100 outline-none focus-visible:ring-3 focus-visible:ring-ring/30 transition-opacity duration-200 z-10"
        >
          {collapsed ? <ArrowRight2 className="size-4" /> : <ArrowLeft2 className="size-4" />}
        </button>
      )}
    </div>
  );
};
