import { Link } from "react-router";
import { Box } from "iconsax-reactjs";
import type { TopProduct } from "@/shared/api";
import { formatNumber, formatPrice } from "@/shared/lib";
import {
  Button,
  Card,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Progress,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui";

function NoSales() {
  return (
    <Empty className="rounded-none border-0 border-t border-solid border-border py-12">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Box className="size-5 text-primary" />
        </EmptyMedia>
        <EmptyTitle>Продаж пока нет</EmptyTitle>
        <EmptyDescription>
          Топ появится, когда первые заказы будут доставлены. Начните с товаров: опубликуйте хотя бы один.
        </EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button nativeButton={false} render={<Link to="/products" />}>
          К товарам
        </Button>
      </EmptyContent>
    </Empty>
  );
}

function TopTable({ products }: { products: TopProduct[] }) {
  // The bar shows each product against the best seller, so the leader is always full.
  const leader = Math.max(...products.map((p) => p.quantity_sold), 1);

  return (
    <Table className="min-w-[560px]">
      <TableHeader>
        <TableRow>
          <TableHead className="w-10 pl-5">#</TableHead>
          <TableHead>Товар</TableHead>
          <TableHead className="text-right">Продано</TableHead>
          <TableHead className="w-1/4">Доля</TableHead>
          <TableHead className="pr-5 text-right">Выручка</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {products.map((product, index) => (
          <TableRow key={product.product_id}>
            <TableCell className="pl-5 text-muted-foreground tabular-nums">{index + 1}</TableCell>
            <TableCell>
              <div className="flex items-center gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                  <Box className="size-5" />
                </div>
                <span className="font-medium">{product.title}</span>
              </div>
            </TableCell>
            <TableCell className="text-right whitespace-nowrap tabular-nums">
              {formatNumber(product.quantity_sold)} шт.
            </TableCell>
            <TableCell>
              <Progress
                value={(product.quantity_sold / leader) * 100}
                aria-label={`Доля от лидера: ${product.title}`}
                className="*:h-2"
              />
            </TableCell>
            <TableCell className="pr-5 text-right font-medium whitespace-nowrap tabular-nums">
              {formatPrice(product.revenue)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

/** Best sellers of the last 30 days by units in delivered orders, at most five. */
export const TopProducts = ({ products }: { products: TopProduct[] }) => {
  return (
    <section aria-labelledby="top-products-title">
      <Card className="gap-0 py-0 shadow-none">
        <div className="flex flex-wrap items-center justify-between gap-3 px-5 pt-5 pb-3">
          <div>
            <h2 id="top-products-title" className="text-base font-semibold">
              Топ-5 товаров за 30 дней
            </h2>
            <p className="text-xs text-muted-foreground">По доставленным заказам, по числу проданных штук</p>
          </div>
          {products.length > 0 && (
            <Button variant="link" className="h-auto p-0" nativeButton={false} render={<Link to="/products" />}>
              Все товары →
            </Button>
          )}
        </div>
        {products.length > 0 ? <TopTable products={products} /> : <NoSales />}
      </Card>
    </section>
  );
};
