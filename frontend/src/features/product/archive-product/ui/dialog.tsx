import { useMutation, useQueryClient } from "@tanstack/react-query";
import { productKeys, type Product } from "@/entities/product";
import { getProducts } from "@/shared/api";
import { getApiErrorMessage } from "@/shared/api";
import { ConfirmDialog } from "@/shared/ui";
import { useArchiveProduct } from "../model/dialog.store";

const api = getProducts();

export const ArchiveProductDialog = () => {
  const { product, close } = useArchiveProduct();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (target: Product) => api.postProductsIdArchive(target.id),
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
      title="Архивировать товар?"
      description={
        product && (
          <>
            «{product.title}» пропадёт из новых заказов, история останется. Вернуть товар из
            архива пока нельзя.
          </>
        )
      }
      confirmLabel="Архивировать"
      variant="destructive"
      pending={mutation.isPending}
      error={mutation.error ? getApiErrorMessage(mutation.error) : null}
      onConfirm={() => product && mutation.mutate(product)}
    />
  );
};
