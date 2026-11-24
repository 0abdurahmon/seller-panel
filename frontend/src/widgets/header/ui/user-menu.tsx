import { ShopAvatar, useShop } from '@/entities/shop';
import { LogoutMenuItem } from '@/features/auth/logout';
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui';

/** The avatar in the header: who is signed in, and account actions. */
export const UserMenu = () => {
  const { data: shop } = useShop();
  const name = shop?.name ?? 'Мой магазин';

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            className="size-10 shrink-0 rounded-xl p-0"
            aria-label={`Меню аккаунта: ${name}`}
          />
        }
      >
        <ShopAvatar shop={shop} />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-56">
        <div className="flex items-center gap-2.5 px-2 py-1.5">
          <ShopAvatar shop={shop} />
          <div className="min-w-0">
            <p className="truncate text-sm leading-tight font-medium">{name}</p>
            <p className="text-xs leading-tight text-muted-foreground">Продавец</p>
          </div>
        </div>
        <DropdownMenuSeparator />
        <LogoutMenuItem />
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
