import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCategoryOptions } from "@/entities/category";
import {
  ProductForm,
  productKeys,
  productToForm,
  type Product,
  type ProductFormValues,
} from "@/entities/product";
import { getProducts, multipart, parseApiError } from "@/shared/api";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui";
import { useEditProduct } from "../model/modal.store";

const api = getProducts();

function EditProductForm({ product, onDone }: { product: Product; onDone: () => void }) {
  const queryClient = useQueryClient();
  const categories = useCategoryOptions();

  const update = useMutation({
    // Stock is left out: PUT does not change it, PATCH /products/{id}/stock does.
    mutationFn: ({ stock: _stock, photos, ...values }: ProductFormValues) => {
      const keep = photos.flatMap((photo) => (photo.url ? [photo.url] : []));
      const added = photos.flatMap((photo) => (photo.file ? [photo.file] : []));
      return api.putProductsId(
        product.id,
        // An empty keep_photos value removes every current photo; a missing field would keep them.
        { ...values, keep_photos: keep.length ? keep : [""], photos: added },
        multipart,
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: productKeys.all });
      onDone();
    },
  });

  const apiError = update.error ? parseApiError(update.error) : null;
  const skuTaken = apiError?.code === "sku_taken";

  return (
    <ProductForm
      defaultValues={productToForm(product)}
      categories={categories.options}
      categoriesStatus={categories.status}
      withStock={false}
      submitLabel="Сохранить"
      pending={update.isPending}
      errorMessage={apiError && !skuTaken ? apiError.message : null}
      fieldErrors={skuTaken ? { sku: apiError.message } : undefined}
      onSubmit={(values) => update.mutate(values)}
      onCancel={onDone}
    />
  );
}

export const EditProductModal = () => {
  const { product, close } = useEditProduct();

  return (
    <Dialog open={product !== null} onOpenChange={(open) => !open && close()}>
      <DialogContent className="max-w-2xl!">
        <DialogHeader>
          <DialogTitle>Редактирование товара</DialogTitle>
          <DialogDescription>
            Остаток меняется отдельно, здесь только данные товара.
          </DialogDescription>
        </DialogHeader>
        {/* Keyed by id, so opening another product starts from its own values. */}
        {product && <EditProductForm key={product.id} product={product} onDone={close} />}
      </DialogContent>
    </Dialog>
  );
};
