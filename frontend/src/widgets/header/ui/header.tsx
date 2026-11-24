import { Fragment } from 'react';
import { Link, useMatches } from 'react-router';
import { useShop } from '@/entities/shop';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/shared/ui';
import { UserMenu } from './user-menu';

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

function PageBreadcrumb({ crumbs, shopName }: { crumbs: Crumb[]; shopName: string }) {
  // On the home page there is no path to show, so the seller is greeted instead.
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

export const Header = () => {
  const crumbs = useCrumbs();
  const title = crumbs[crumbs.length - 1]?.title;
  const { data: shop } = useShop();

  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex flex-col">
        <h1 className="text-base font-medium md:text-2xl md:font-semibold">{title}</h1>
        <PageBreadcrumb crumbs={crumbs} shopName={shop?.name ?? 'Мой магазин'} />
      </div>
      <UserMenu />
    </div>
  );
};
