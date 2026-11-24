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
import { ArrowDown, ArrowUp, Box, More } from "iconsax-reactjs";
import { Skeleton } from "skullet/react";
import { useCategoryName } from "@/entities/category";
import { ProductStatusBadge, StockLabelBadge } from "@/entities/product";
import { ActivateProductMenuItem } from "@/features/product/activate-product";
import { ArchiveProductMenuItem } from "@/features/product/archive-product";
import { EditProductMenuItem } from "@/features/product/edit-product";
import { useProductDetails } from "@/features/product/product-details";
import { formatDateTime, formatPrice } from "@/shared/lib/format";
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
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
import type { Product } from "@/shared/api/model";

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
        <Button variant="outline" onClick={onResetFilters}>
          Сбросить фильтры
        </Button>
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
    <div className="flex justify-end" onClick={(event) => event.stopPropagation()}>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="icon-sm" aria-label={`Действия: ${product.title}`} />
          }
        >
          <More className="rotate-90" />
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
        <button
          type="button"
          className="inline-flex items-center gap-1.5 cursor-pointer select-none hover:text-foreground transition-colors"
          onClick={() => column.toggleSorting(isSorted === "asc")}
        >
          <span>Дата создания</span>
          {isSorted === "asc" ? (
            <ArrowUp className="size-3.5 text-primary" />
          ) : isSorted === "desc" ? (
            <ArrowDown className="size-3.5 text-primary" />
          ) : (
            <ArrowUp className="size-3.5 opacity-30" />
          )}
        </button>
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
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Box className="size-5" />
          </div>
          <div className="min-w-48 space-y-0.5">
            {/* A button so the details open from the keyboard too; a mouse click anywhere on the row does the same. */}
            <button
              type="button"
              onClick={() => useProductDetails.getState().openProduct(product)}
              className="rounded-md text-left font-medium outline-none hover:text-primary focus-visible:ring-3 focus-visible:ring-ring/30"
            >
              {product.title}
            </button>
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
  columnHelper.accessor("available", {
    id: "stock",
    header: "Доступно",
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <span className="tabular-nums">{row.original.available}</span>
        <StockLabelBadge label={row.original.stock_label} />
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
  available: 24,
  stock_label: "in_stock",
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
  return (
    <Skeleton
      id="products-table"
      label="Загрузка товаров"
      data={isLoading ? null : products}
      sample={SAMPLE_PRODUCTS}
      empty={!isLoading && products?.length === 0}
      emptyMessage={<ProductsEmptyState onResetFilters={onResetFilters} />}
    >
      {(rows) => <ProductsTableView products={rows} />}
    </Skeleton>
  );
}
