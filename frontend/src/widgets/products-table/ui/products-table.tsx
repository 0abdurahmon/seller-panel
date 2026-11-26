import { useState } from "react";
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, Box, Sort } from "iconsax-reactjs";
import { Skeleton } from "skullet/react";
import { useCategoryName } from "@/entities/category";
import {
  getAvailable,
  getStockLabel,
  ProductImage,
  ProductStatusBadge,
  StockLabelBadge,
} from "@/entities/product";
import { ActivateProductMenuItem } from "@/features/product/activate-product";
import { ArchiveProductMenuItem } from "@/features/product/archive-product";
import { EditProductMenuItem } from "@/features/product/edit-product";
import { useProductDetails } from "@/features/product/product-details";
import { RestockProductButton } from "@/features/product/restock-product";
import { formatDateTime, formatPrice } from "@/shared/lib";
import {
  Button,
  MoreVerticalIcon,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui";
import type { Product } from "@/shared/api";

function ProductsEmptyState({ onResetFilters }: { onResetFilters?: () => void }) {
  // With filters on, "nothing found" is about the filters, not about an empty shop.
  const filtered = Boolean(onResetFilters);

  return (
    <Empty className="flex-1 h-full min-h-0 w-full rounded-3xl border border-dashed border-border bg-card p-12">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Box className="size-5 text-primary" />
        </EmptyMedia>
        <EmptyTitle>{filtered ? "Ничего не найдено" : "Товары не найдены"}</EmptyTitle>
        <EmptyDescription>
          {filtered
            ? "Ни один товар не подходит под поиск и фильтры."
            : "Попробуйте другую вкладку или добавьте товар."}
        </EmptyDescription>
      </EmptyHeader>
      {filtered && (
        <EmptyContent>
          <Button variant="outline" onClick={onResetFilters}>
            Сбросить фильтры
          </Button>
        </EmptyContent>
      )}
    </Empty>
  );
}

function CategoryName({ id }: { id: number }) {
  const categoryName = useCategoryName();
  return <span className="whitespace-nowrap">{categoryName(id)}</span>;
}

function ProductActions({ product }: { product: Product }) {
  // An archived product has no actions yet, so it gets no menu.
  if (product.status === "archived") return null;

  return (
    // Clicks inside the menu (it is portaled, but React events still bubble) must not open the row's details.
    <div className="flex items-center justify-end gap-1" onClick={(event) => event.stopPropagation()}>
      <RestockProductButton product={product} />
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon"
              className="hover:bg-foreground/10 dark:hover:bg-foreground/15 [&_svg:not([class*='size-'])]:size-5"
              aria-label={`Действия: ${product.title}`}
            />
          }
        >
          <MoreVerticalIcon />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <EditProductMenuItem product={product} />
          <ActivateProductMenuItem product={product} />
          <DropdownMenuSeparator />
          <ArchiveProductMenuItem product={product} />
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

const columnHelper = createColumnHelper<Product>();

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- column value types differ per column
const columns: ColumnDef<Product, any>[] = [
  columnHelper.accessor("created_at", {
    id: "created_at",
    header: ({ column }) => {
      const isSorted = column.getIsSorted();
      return (
        <Button
          variant="ghost"
          size="sm"
          className="-ml-3 text-muted-foreground"
          onClick={() => column.toggleSorting(isSorted === "asc")}
        >
          Дата создания
          {isSorted === "asc" ? (
            <ArrowUp className="text-primary" />
          ) : isSorted === "desc" ? (
            <ArrowDown className="text-primary" />
          ) : (
            <Sort />
          )}
        </Button>
      );
    },
    cell: ({ getValue }) => (
      <span className="text-sm text-muted-foreground whitespace-nowrap">
        {formatDateTime(getValue())}
      </span>
    ),
  }),
  columnHelper.accessor("title", {
    id: "product",
    header: "Товар",
    cell: ({ row }) => {
      const product = row.original;
      return (
        <div className="flex items-center gap-3">
          <ProductImage product={product} />
          <div className="min-w-48 space-y-0.5">
            {/* A button so the details open from the keyboard too; a mouse click anywhere on the row does the same. */}
            <Button
              variant="link"
              onClick={() => useProductDetails.getState().openProduct(product)}
              className="h-auto p-0 text-foreground hover:text-primary hover:no-underline"
            >
              {product.title}
            </Button>
            <p className="text-xs text-muted-foreground">{product.sku}</p>
          </div>
        </div>
      );
    },
  }),
  columnHelper.accessor("category_id", {
    id: "category",
    header: "Категория",
    cell: ({ getValue }) => <CategoryName id={getValue()} />,
  }),
  columnHelper.accessor("price", {
    id: "price",
    header: () => <div className="text-right">Цена</div>,
    cell: ({ getValue }) => (
      <div className="text-right whitespace-nowrap tabular-nums">
        {formatPrice(getValue())}
      </div>
    ),
  }),
  columnHelper.accessor((row) => getAvailable(row), {
    id: "stock",
    header: "Доступно",
    cell: ({ row, getValue }) => (
      <div className="flex items-center gap-2">
        <span className="tabular-nums">{getValue()}</span>
        <StockLabelBadge label={getStockLabel(row.original)} />
      </div>
    ),
  }),
  columnHelper.accessor("status", {
    id: "status",
    header: "Статус",
    cell: ({ getValue }) => <ProductStatusBadge status={getValue()} />,
  }),
  columnHelper.display({
    id: "actions",
    header: () => <span className="sr-only">Действия</span>,
    cell: ({ row }) => <ProductActions product={row.original} />,
  }),
];

// Stands in for real rows while loading, so skullet can measure the table's shape.
// Lengths are close to real data on purpose.
const SAMPLE_PRODUCTS: Product[] = Array.from({ length: 5 }, (_, i) => ({
  id: -1 - i,
  shop_id: 0,
  title: "Смартфон Galaxy A55 8/256 ГБ",
  description: "",
  sku: "PH-A55-256",
  category_id: 0,
  price: 4_890_000,
  stock: 24,
  reserved: 0,
  photos: [],
  status: "active",
  created_at: "2026-11-24T09:12:00Z",
  updated_at: "2026-11-24T09:12:00Z",
}));

function ProductsTableView({ products }: { products: Product[] }) {
  const openProduct = useProductDetails((state) => state.openProduct);
  const [sorting, setSorting] = useState<SortingState>([
    { id: "created_at", desc: true },
  ]);

  const table = useReactTable({
    data: products,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="overflow-hidden rounded-3xl border border-border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id} className="first:pl-5 last:pr-5">
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow
              key={row.id}
              className="cursor-pointer"
              onClick={() => openProduct(row.original)}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id} className="first:pl-5 last:pr-5">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function ProductsTable({
  products,
  isLoading,
  onResetFilters,
}: {
  products: Product[] | undefined;
  isLoading?: boolean;
  /** Set when search or filters are on; the empty state then offers to reset them. */
  onResetFilters?: () => void;
}) {
  const isEmpty = !isLoading && products?.length === 0;

  return (
    <Skeleton
      id="products-table"
      // Skullet wraps the content in a pane: when empty, both stretch so the empty state
      // fills the page; with rows the table keeps its own height.
      className={isEmpty ? "flex flex-1 flex-col *:flex *:flex-1 *:flex-col" : undefined}
      label="Загрузка товаров"
      data={isLoading ? null : products}
      sample={SAMPLE_PRODUCTS}
      empty={isEmpty}
      emptyMessage={<ProductsEmptyState onResetFilters={onResetFilters} />}
    >
      {(rows) => <ProductsTableView products={rows} />}
    </Skeleton>
  );
}
