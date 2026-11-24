import type { ReactNode } from "react";
import { Box } from "iconsax-reactjs";
import { useCategoryName } from "@/entities/category";
import {
  ProductStatusBadge,
  StockLabelBadge,
  type Product,
} from "@/entities/product";
import { formatDateTime, formatPrice } from "@/shared/lib/format";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui";
import { useProductDetails } from "../model/modal.store";

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-sm font-medium">{children}</dd>
    </div>
  );
}

function ProductDetails({ product }: { product: Product }) {
  const categoryName = useCategoryName();

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-2">
        <ProductStatusBadge status={product.status} />
        <StockLabelBadge label={product.stock_label} />
      </div>

      <dl className="grid grid-cols-2 gap-4 rounded-2xl bg-muted/50 p-4 sm:grid-cols-4">
        <Detail label="Цена">
          <span className="tabular-nums">{formatPrice(product.price)}</span>
        </Detail>
        <Detail label="Всего на складе">
          <span className="tabular-nums">{product.stock}</span>
        </Detail>
        <Detail label="В резерве">
          <span className="tabular-nums">{product.reserved}</span>
        </Detail>
        <Detail label="Доступно">
          <span className="tabular-nums">{product.available}</span>
        </Detail>
      </dl>

      <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Detail label="Категория">{categoryName(product.category_id)}</Detail>
        <Detail label="Создан">{formatDateTime(product.created_at)}</Detail>
        <Detail label="Изменён">{formatDateTime(product.updated_at)}</Detail>
      </dl>

      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Описание</p>
        {product.description ? (
          <p className="text-sm whitespace-pre-line">{product.description}</p>
        ) : (
          <p className="text-sm text-muted-foreground">Описание не заполнено</p>
        )}
      </div>
    </div>
  );
}

export const ProductDetailsModal = () => {
  const { product, close } = useProductDetails();

  return (
    <Dialog open={product !== null} onOpenChange={(open) => !open && close()}>
      <DialogContent className="max-w-2xl!">
        {product && (
          <>
            <DialogHeader>
              <div className="flex items-center gap-4">
                <div className="flex size-16 shrink-0 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
                  <Box className="size-7" />
                </div>
                <div className="min-w-0 space-y-1">
                  <DialogTitle>{product.title}</DialogTitle>
                  <DialogDescription>Артикул: {product.sku}</DialogDescription>
                </div>
              </div>
            </DialogHeader>
            <ProductDetails product={product} />
          </>
        )}
      </DialogContent>
    </Dialog>
  );
};
