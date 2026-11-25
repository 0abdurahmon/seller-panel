import { Link, useMatch } from 'react-router';
import { cn } from 'cn';
import type { MenuItem } from '@/shared/config';
import { Badge, Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui';

export function MenuLink({
  item,
  collapsed,
  onNavigate,
  badge,
}: {
  item: MenuItem;
  collapsed: boolean;
  onNavigate?: () => void;
  /** A count shown next to the label, for example orders waiting for action. Hidden when 0. */
  badge?: number;
}) {
  // "/" is active only on the home page; other items stay active on nested pages (/products/42).
  const isActive = Boolean(useMatch({ path: item.href, end: item.href === '/' }));
  const ItemIcon = item.icon;

  const element = (
    <Link
      to={item.href}
      onClick={onNavigate}
      aria-current={isActive ? 'page' : undefined}
      aria-label={badge ? `${item.label}: ${badge}` : undefined}
      className={cn(
        'relative font-medium text-sm flex items-center justify-start py-3 rounded-xl transition-colors duration-200 overflow-hidden whitespace-nowrap w-full text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/30',
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
      {badge ? (
        collapsed ? (
          <span aria-hidden className="absolute top-2 right-2 size-2 rounded-full bg-primary ring-2 ring-sidebar" />
        ) : (
          <Badge variant={isActive ? 'secondary' : 'default'} className="tabular-nums">
            {badge}
          </Badge>
        )
      ) : null}
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
