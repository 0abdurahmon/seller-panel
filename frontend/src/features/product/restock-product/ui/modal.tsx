import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { productKeys, type Product } from "@/entities/product";
import { getApiErrorMessage, getStock } from "@/shared/api";
import {
  Alert,
  AlertDescription,
  Button,
  Credenza,
  CredenzaContent,
  CredenzaDescription,
  CredenzaHeader,
  CredenzaTitle,
  Field,
  FieldDescription,
  FieldLabel,
  NumberStepper,
  Spinner,
} from "@/shared/ui";
import { useRestockProduct } from "../model/modal.store";

const api = getStock();

function RestockProductForm({ product, onDone }: { product: Product; onDone: () => void }) {
  const [stock, setStock] = useState(product.stock);
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (nextStock: number) =>
      api.patchProductsIdStock(product.id, {
        stock: nextStock,
        expected_stock: product.stock,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: productKeys.all });
      onDone();
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (stock === product.stock || mutation.isPending) return;
    mutation.mutate(stock);
  };

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      <Field>
        <FieldLabel>Остаток, шт.</FieldLabel>
        <NumberStepper
          value={stock}
          onChange={setStock}
          min={product.reserved}
          max={1_000_000}
          decrementLabel="Уменьшить остаток"
          incrementLabel="Увеличить остаток"
        />
        <FieldDescription>
          {product.reserved > 0
            ? `Сейчас на складе: ${product.stock.toLocaleString("ru-RU")} шт., в резерве: ${product.reserved.toLocaleString("ru-RU")} шт.`
            : `Сейчас на складе: ${product.stock.toLocaleString("ru-RU")} шт.`}
        </FieldDescription>
      </Field>

      {mutation.error && (
        <Alert variant="destructive">
          <AlertDescription>{getApiErrorMessage(mutation.error)}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-end gap-2">
        <Button type="button" variant="ghost" size="lg" onClick={onDone}>
          Отмена
        </Button>
        <Button
          type="submit"
          size="lg"
          disabled={stock === product.stock || mutation.isPending}
        >
          {mutation.isPending && <Spinner />}
          Сохранить
        </Button>
      </div>
    </form>
  );
}

export const RestockProductModal = () => {
  const { product, close } = useRestockProduct();

  return (
    <Credenza open={product !== null} onOpenChange={(open) => !open && close()}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Изменение остатка</CredenzaTitle>
          <CredenzaDescription>{product?.title}</CredenzaDescription>
        </CredenzaHeader>
        {/* Keyed by id and stock so each open starts from the latest observed stock. */}
        {product && (
          <RestockProductForm
            key={`${product.id}-${product.stock}`}
            product={product}
            onDone={close}
          />
        )}
      </CredenzaContent>
    </Credenza>
  );
};
