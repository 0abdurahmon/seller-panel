import { useSearchParams } from "react-router";
import {
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
import {
  CategoryFilter,
  ProductSearch,
} from "@/features/product/filter-products";
import { ProductDetailsModal } from "@/features/product/product-details";
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

type ParamsUpdate = {
  status?: ProductStatusFilter;
  page?: number;
  q?: string;
  category?: number | null;
};

export const ProductsPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const status = parseStatus(searchParams.get("status"));
  const page = parsePage(searchParams.get("page"));
  const q = searchParams.get("q") ?? "";
  const category = parseCategory(searchParams.get("category"));
  const hasFilters = q !== "" || category !== null;

  function updateParams(next: ParamsUpdate) {
    const params = new URLSearchParams(searchParams);
    const setOrDelete = (key: string, value: string | null) =>
      value ? params.set(key, value) : params.delete(key);

    if (next.status !== undefined) setOrDelete("status", next.status === "all" ? null : next.status);
    if (next.q !== undefined) setOrDelete("q", next.q);
    if (next.category !== undefined) setOrDelete("category", next.category ? String(next.category) : null);
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

  return (
    <div className="flex flex-1 flex-col gap-6">
      <CreateProductModal />
      <ProductDetailsModal />
      <EditProductModal />
      <ActivateProductDialog />
      <ArchiveProductDialog />

      <div className="flex flex-wrap items-center justify-between gap-4">
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
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <ProductSearch
          value={q}
          // The same search again (a trailing space was typed) must not reset the page.
          onChange={(next) => next !== q && updateParams({ q: next })}
        />
        <CategoryFilter
          value={category}
          onChange={(next) => updateParams({ category: next })}
        />
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
            products={products.data?.items}
            isLoading={products.isPending}
            onResetFilters={
              hasFilters ? () => updateParams({ q: "", category: null }) : undefined
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
