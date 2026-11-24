import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCategoryOptions } from "@/entities/category";
import {
  emptyProductForm,
  ProductForm,
  productKeys,
  type ProductFormValues,
} from "@/entities/product";
import { getProducts } from "@/shared/api";
import { parseApiError } from "@/shared/api";

const api = getProducts();

export const CreateProductForm = ({ onDone }: { onDone: () => void }) => {
  const queryClient = useQueryClient();
  const categories = useCategoryOptions();

  const create = useMutation({
    mutationFn: (values: ProductFormValues) => api.postProducts(values),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: productKeys.all });
      onDone();
    },
  });

  const apiError = create.error ? parseApiError(create.error) : null;
  const skuTaken = apiError?.code === "sku_taken";

  return (
    <ProductForm
      defaultValues={emptyProductForm}
      categories={categories.options}
      categoriesStatus={categories.status}
      withStock
      submitLabel="Создать"
      pending={create.isPending}
      errorMessage={apiError && !skuTaken ? apiError.message : null}
      fieldErrors={skuTaken ? { sku: apiError.message } : undefined}
      onSubmit={(values) => create.mutate(values)}
      onCancel={onDone}
    />
  );
};
