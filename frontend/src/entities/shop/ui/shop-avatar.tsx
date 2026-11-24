import { cn } from 'cn';
import type { Shop } from '@/shared/api';
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui';

function getInitials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase())
    .join('');
}

/** The shop's logo, or its initials when there is none. */
export const ShopAvatar = ({
  shop,
  className,
}: {
  shop: Pick<Shop, 'name' | 'logo_url'> | undefined;
  className?: string;
}) => {
  return (
    <Avatar className={cn('size-10 rounded-xl', className)}>
      {shop?.logo_url ? <AvatarImage src={shop.logo_url} alt="" /> : null}
      <AvatarFallback className="rounded-xl bg-muted text-foreground">
        {shop ? getInitials(shop.name) : null}
      </AvatarFallback>
    </Avatar>
  );
};
