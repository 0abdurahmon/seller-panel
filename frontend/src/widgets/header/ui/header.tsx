import { Fragment } from 'react';
import { Link, useMatches } from 'react-router';
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/shared/ui';

/** Route `handle` shape: a route sets `handle: { title: '...' }` to name its page. */
export type RouteHandle = { title?: string };

type Crumb = { title: string; href: string };

const HOME: Crumb = { title: 'Главная', href: '/' };

/** Every matched route with a title, from the top level down to the current page. */
function useCrumbs(): Crumb[] {
  const matches = useMatches();
  const crumbs = matches.flatMap((match) => {
    const title = (match.handle as RouteHandle | undefined)?.title;
    return title ? [{ title, href: match.pathname }] : [];
  });
  return crumbs[0]?.href === '/' ? crumbs : [HOME, ...crumbs];
}

function getInitials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase())
    .join('');
}

function PageBreadcrumb({ crumbs, shopName }: { crumbs: Crumb[]; shopName: string }) {
  // On the home page there is no path to show; greet the seller instead, as tutoria does.
  if (crumbs.length <= 1) {
    return <p className="text-sm text-muted-foreground">Добро пожаловать, {shopName}</p>;
  }

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {crumbs.map((crumb, index) => {
          const isLast = index === crumbs.length - 1;
          return (
            <Fragment key={crumb.href}>
              {index > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{crumb.title}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink className="text-primary" render={<Link to={crumb.href} />}>
                    {crumb.title}
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}

// Replace with the shop from the API once auth is connected.
const FALLBACK_SHOP_NAME = 'Мой магазин';

export const Header = ({
  shopName = FALLBACK_SHOP_NAME,
  logoUrl,
}: {
  shopName?: string;
  logoUrl?: string | null;
}) => {
  const crumbs = useCrumbs();
  const title = crumbs[crumbs.length - 1]?.title;

  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex flex-col">
        <h1 className="text-base font-medium md:text-2xl md:font-semibold">{title}</h1>
        <PageBreadcrumb crumbs={crumbs} shopName={shopName} />
      </div>
      <Link
        to="/shop"
        aria-label={`Магазин: ${shopName}`}
        title={shopName}
        className="shrink-0 rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
      >
        <Avatar className="size-10 rounded-xl">
          {logoUrl ? <AvatarImage src={logoUrl} alt="" /> : null}
          <AvatarFallback className="bg-muted text-foreground">
            {getInitials(shopName)}
          </AvatarFallback>
        </Avatar>
      </Link>
    </div>
  );
};
