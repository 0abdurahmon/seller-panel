import { useRef } from 'react';
import { Link, useLocation } from 'react-router';
import { cn } from 'cn';
import { useDashboardSummary } from '@/entities/dashboard';
import { bottomMenuItems, mainMenuItems } from '@/shared/config';
import { Logo, LogoMark } from '@/shared/ui';
import { useActivePill } from '../model/use-active-pill';
import { MenuLink } from './menu-link';

/** Logo and navigation links: the content of the desktop sidebar and of the phone menu. */
export function AsideNav({
  collapsed,
  onNavigate,
}: {
  collapsed: boolean;
  /** Called when a link is followed, so the phone menu can close. */
  onNavigate?: () => void;
}) {
  // Shares the dashboard's cached summary, so it costs no extra request.
  const waiting = useDashboardSummary().data?.waiting_for_action;
  const { pathname } = useLocation();
  const navRef = useRef<HTMLElement>(null);
  const pillRef = useRef<HTMLSpanElement>(null);

  useActivePill(navRef, pillRef, pathname);

  return (
    <>
      <Link
        to="/"
        aria-label="Dona"
        onClick={onNavigate}
        className={cn(
          'mb-6 flex h-8 items-center rounded-xl text-foreground outline-none focus-visible:ring-3 focus-visible:ring-ring/30',
          collapsed ? 'justify-center' : 'px-3.5',
        )}
      >
        {collapsed ? <LogoMark className="size-8 shrink-0" /> : <Logo className="h-8 w-auto" />}
      </Link>
      <nav ref={navRef} aria-label="Основное меню" className="relative flex flex-1 flex-col">
        <span
          ref={pillRef}
          aria-hidden
          className="pointer-events-none absolute inset-x-0 top-0 z-0 rounded-xl bg-primary opacity-0 will-change-[transform,height] dark:bg-sidebar-accent"
        />
        <ul className="space-y-1">
          {mainMenuItems.map((item) => (
            <MenuLink
              key={item.href}
              item={item}
              collapsed={collapsed}
              onNavigate={onNavigate}
              badge={item.href === '/orders' ? waiting : undefined}
            />
          ))}
        </ul>
        <ul className="mt-auto space-y-1">
          {bottomMenuItems.map((item) => (
            <MenuLink key={item.href} item={item} collapsed={collapsed} onNavigate={onNavigate} />
          ))}
        </ul>
      </nav>
    </>
  );
}
