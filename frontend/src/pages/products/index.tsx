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
import { ProductDetailsModal } from "@/features/product/product-details";
import { getApiErrorMessage } from "@/shared/api/http/error";
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

export const ProductsPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const status = parseStatus(searchParams.get("status"));
  const page = parsePage(searchParams.get("page"));

  function updateParams(next: { status?: ProductStatusFilter; page?: number }) {
    const params = new URLSearchParams(searchParams);
    if (next.status !== undefined) {
      if (next.status === "all") params.delete("status");
      else params.set("status", next.status);
      // A new tab starts from its first page.
      params.delete("page");
    }
    if (next.page !== undefined) {
      if (next.page <= 1) params.delete("page");
      else params.set("page", String(next.page));
    }
    setSearchParams(params, { replace: true });
  }

  const products = useProducts({
    page,
    page_size: PAGE_SIZE,
    ...(status === "all" ? {} : { status }),
  });
  const counts = useProductCounts();

  return (
    <div className="flex min-h-full flex-col space-y-6">
      <CreateProductModal />
      <ProductDetailsModal />

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
