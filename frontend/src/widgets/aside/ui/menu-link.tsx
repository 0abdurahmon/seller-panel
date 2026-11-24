import { Link, useMatch } from 'react-router';
import { cn } from 'cn';
import type { MenuItem } from '@/shared/config';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui';

export function MenuLink({
  item,
  collapsed,
  onNavigate,
}: {
  item: MenuItem;
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  // "/" is active only on the home page; other items stay active on nested pages (/products/42).
  const isActive = Boolean(useMatch({ path: item.href, end: item.href === '/' }));
  const ItemIcon = item.icon;

  const element = (
    <Link
      to={item.href}
      onClick={onNavigate}
      aria-current={isActive ? 'page' : undefined}
      className={cn(
        'font-medium text-sm flex items-center justify-start py-3 rounded-xl transition-colors duration-200 overflow-hidden whitespace-nowrap w-full text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/30',
        collapsed ? 'px-3.5' : 'px-3.5 gap-x-3',
        isActive
          ? 'bg-primary text-primary-foreground dark:bg-sidebar-accent dark:text-sidebar-accent-foreground'
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
