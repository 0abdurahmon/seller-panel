import { useState } from 'react';
import { Button, MenuIcon, Sheet, SheetContent, SheetTitle, SheetTrigger } from '@/shared/ui';
import { AsideNav } from './aside-nav';

/** Phones only: a menu button that slides the sidebar in from the left. */
export const MobileMenu = () => {
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        render={
          <Button
            variant="outline"
            size="icon-lg"
            className="shrink-0 rounded-xl md:hidden"
            aria-label="Открыть меню"
          />
        }
      >
        <MenuIcon />
      </SheetTrigger>
      <SheetContent
        side="left"
        className="w-64 border-sidebar-border bg-sidebar px-2 py-6 data-[side=left]:w-64"
      >
        <SheetTitle className="sr-only">Меню</SheetTitle>
        <AsideNav collapsed={false} onNavigate={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  );
};
