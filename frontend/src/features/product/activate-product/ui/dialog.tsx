import { useMutation, useQueryClient } from "@tanstack/react-query";
import { productKeys, type Product } from "@/entities/product";
import { getProducts } from "@/shared/api";
import { getApiErrorMessage } from "@/shared/api";
import { ConfirmDialog } from "@/shared/ui";
import { useActivateProduct } from "../model/dialog.store";

const api = getProducts();

// There is no activate endpoint: PUT replaces the editable fields, so it sends them unchanged with the new status.
function activate(product: Product) {
  return api.putProductsId(product.id, {
    category_id: product.category_id,
    title: product.title,
    description: product.description,
    price: product.price,
    sku: product.sku,
    status: "active",
  });
}

export const ActivateProductDialog = () => {
  const { product, close } = useActivateProduct();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: activate,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: productKeys.all });
      close();
    },
  });

  return (
    <ConfirmDialog
      open={product !== null}
      onOpenChange={(open) => {
        if (open) return;
        mutation.reset();
        close();
      }}
      title="Активировать товар?"
      description={
        product && <>«{product.title}» станет активным, и покупатели смогут его заказать.</>
      }
      confirmLabel="Активировать"
      pending={mutation.isPending}
      error={mutation.error ? getApiErrorMessage(mutation.error) : null}
      onConfirm={() => product && mutation.mutate(product)}
    />
  );
};
