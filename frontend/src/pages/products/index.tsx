import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { cn } from "cn";
import {
  getStockLabel,
  useProductCounts,
  useProducts,
  type ProductStatusFilter,
} from "@/entities/product";
import {
  CreateProductButton,
  CreateProductModal,
} from "@/features/product/create-product";
import { ActivateProductDialog } from "@/features/product/activate-product";
import { ArchiveProductDialog } from "@/features/product/archive-product";
import { EditProductModal } from "@/features/product/edit-product";
import { ProductFiltersButton, type StockFilter } from "@/features/product/filter-products";
import { ProductDetailsModal } from "@/features/product/product-details";
import { RestockProductModal } from "@/features/product/restock-product";
import { getApiErrorMessage } from "@/shared/api";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
  Tabs,
  TabsCount,
  TabsList,
  TabsTrigger,
} from "@/shared/ui";
import { ProductsPagination, ProductsTable } from "@/widgets/products-table";

const PAGE_SIZE = 20;

const STATUS_TABS: { value: ProductStatusFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "active", label: "Активные" },
  { value: "draft", label: "Черновики" },
  { value: "archived", label: "В архиве" },
];

function getScrollParent(node: HTMLElement) {
  for (let el = node.parentElement; el; el = el.parentElement) {
    if (/auto|scroll/.test(getComputedStyle(el).overflowY)) return el;
  }
  return null;
}

function parseStatus(value: string | null): ProductStatusFilter {
  return STATUS_TABS.some((tab) => tab.value === value)
    ? (value as ProductStatusFilter)
    : "all";
}

function parsePage(value: string | null) {
  const page = Number(value);
  return Number.isInteger(page) && page >= 1 ? page : 1;
}

function parseCategory(value: string | null) {
  const id = Number(value);
  return value && Number.isInteger(id) && id >= 1 ? id : null;
}

const STOCK_FILTERS: StockFilter[] = ["all", "in_stock", "low_stock", "out_of_stock"];

function parseStock(value: string | null): StockFilter {
  return STOCK_FILTERS.includes(value as StockFilter) ? (value as StockFilter) : "all";
}

type ParamsUpdate = {
  status?: ProductStatusFilter;
  page?: number;
  q?: string;
  category?: number | null;
  stock?: StockFilter;
};

export const ProductsPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const status = parseStatus(searchParams.get("status"));
  const page = parsePage(searchParams.get("page"));
  const q = searchParams.get("q") ?? "";
  const category = parseCategory(searchParams.get("category"));
  const stock = parseStock(searchParams.get("stock"));
  const hasFilters = q !== "" || category !== null || status !== "all" || stock !== "all";

  function updateParams(next: ParamsUpdate) {
    const params = new URLSearchParams(searchParams);
    const setOrDelete = (key: string, value: string | null) =>
      value ? params.set(key, value) : params.delete(key);

    if (next.status !== undefined) setOrDelete("status", next.status === "all" ? null : next.status);
    if (next.q !== undefined) setOrDelete("q", next.q);
    if (next.category !== undefined) setOrDelete("category", next.category ? String(next.category) : null);
    if (next.stock !== undefined) setOrDelete("stock", next.stock === "all" ? null : next.stock);
    // Any new filter starts from the first page; only an explicit page change keeps one.
    setOrDelete("page", next.page && next.page > 1 ? String(next.page) : null);

    setSearchParams(params, { replace: true });
  }

  const filters = {
    ...(q ? { q } : {}),
    ...(category ? { category } : {}),
  };
  const products = useProducts({
    ...filters,
    page,
    page_size: PAGE_SIZE,
    ...(status === "all" ? {} : { status }),
  });
  const counts = useProductCounts(filters);

  const stickySentinel = useRef<HTMLDivElement>(null);
  const [isTabsStuck, setIsTabsStuck] = useState(false);

  useEffect(() => {
    const node = stickySentinel.current;
    if (!node) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]) setIsTabsStuck(!entries[0].isIntersecting);
      },
      { root: getScrollParent(node), threshold: 0 },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return (
    <div className="flex flex-1 flex-col gap-3">
      <CreateProductModal />
      <ProductDetailsModal />
      <EditProductModal />
      <RestockProductModal />
      <ActivateProductDialog />
      <ArchiveProductDialog />

      <div ref={stickySentinel} className="-mb-3 h-px" aria-hidden />
      <div
        className={cn(
          "sticky -top-3 z-20 -mx-5 -mt-3 flex flex-wrap items-center justify-between gap-4 border-b bg-background px-5 py-3 transition-[box-shadow,border-color] duration-200",
          isTabsStuck
            ? "border-border/60 shadow-lg shadow-foreground/5"
            : "border-transparent",
        )}
      >
        <Tabs
          value={status}
          onValueChange={(value) =>
            updateParams({ status: value as ProductStatusFilter })
          }
        >
          <TabsList>
            {STATUS_TABS.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label} <TabsCount count={counts[tab.value]} />
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-2">
          <CreateProductButton />
          <ProductFiltersButton
            category={category}
            status={status}
            stock={stock}
            onCategoryChange={(next) => updateParams({ category: next })}
            onStatusChange={(next) => updateParams({ status: next })}
            onStockChange={(next) => updateParams({ stock: next })}
            onReset={() => updateParams({ q: "", category: null, status: "all", stock: "all" })}
          />
        </div>
      </div>

      {products.isError ? (
        <Alert variant="destructive">
          <AlertTitle>Не удалось загрузить товары</AlertTitle>
          <AlertDescription>{getApiErrorMessage(products.error)}</AlertDescription>
          <AlertAction>
            <Button size="sm" variant="outline" onClick={() => products.refetch()}>
              Повторить
            </Button>
          </AlertAction>
        </Alert>
      ) : (
        <>
          <ProductsTable
            products={
              products.data?.items && stock !== "all"
                ? products.data.items.filter((item) => getStockLabel(item) === stock)
                : products.data?.items
            }
            isLoading={products.isPending}
            onResetFilters={
              hasFilters
                ? () => updateParams({ q: "", category: null, status: "all", stock: "all" })
                : undefined
            }
          />
          {products.data && (
            <ProductsPagination
              page={products.data.page}
              pageSize={products.data.page_size}
              total={products.data.total}
              disabled={products.isPlaceholderData}
              onPageChange={(next) => updateParams({ page: next })}
            />
          )}
        </>
      )}
    </div>
  );
};
