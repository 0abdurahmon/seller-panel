import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { categoryLabel, useCategories } from "@/entities/category";
import { productKeys } from "@/entities/product";
import { getProducts } from "@/shared/api/generated";
import { parseApiError } from "@/shared/api/http/error";
import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldError,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Textarea,
} from "@/shared/ui";
import { formSchema, type FormInput, type FormOutput } from "../model/form-schema";

const api = getProducts();

export const CreateProductForm = ({ onDone }: { onDone: () => void }) => {
  const queryClient = useQueryClient();
  const categories = useCategories();

  const form = useForm<FormInput, unknown, FormOutput>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      category_id: "",
      title: "",
      description: "",
      price: "",
      sku: "",
      stock: "0",
    },
  });

  const create = useMutation({
    mutationFn: (values: FormOutput) => api.postProducts(values),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: productKeys.all });
      onDone();
    },
    onError: (error) => {
      const parsed = parseApiError(error);
      if (parsed.code === "sku_taken") {
        form.setError("sku", { message: "Такой артикул уже есть в магазине" });
      }
    },
  });

  const apiError = create.error ? parseApiError(create.error) : null;
  const showApiError = apiError && apiError.code !== "sku_taken";

  const items = (categories.data ?? []).map((c) => ({
    value: String(c.id),
    label: categoryLabel(c, categories.data ?? []),
  }));

  const { errors } = form.formState;

  return (
    <form className="space-y-3" onSubmit={form.handleSubmit((values) => create.mutate(values))} noValidate>
      <Controller
        name="category_id"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.category_id}>
            <FieldLabel>Категория</FieldLabel>
            <Select
              items={items}
              value={field.value || null}
              onValueChange={(value) => field.onChange(value ?? "")}
              disabled={categories.isPending || categories.isError}
            >
              <SelectTrigger className="w-full" aria-invalid={!!errors.category_id} onBlur={field.onBlur}>
                <SelectValue
                  placeholder={
                    categories.isPending
                      ? "Загрузка категорий…"
                      : categories.isError
                        ? "Не удалось загрузить категории"
                        : "Выберите категорию"
                  }
                />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {items.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldError errors={[errors.category_id]} />
          </Field>
        )}
      />
      <Controller
        name="title"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.title}>
            <FieldLabel>Название</FieldLabel>
            <Input {...field} aria-invalid={!!errors.title} />
            <FieldError errors={[errors.title]} />
          </Field>
        )}
      />
      <Controller
        name="description"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.description}>
            <FieldLabel>Описание</FieldLabel>
            <Textarea {...field} aria-invalid={!!errors.description} />
            <FieldError errors={[errors.description]} />
          </Field>
        )}
      />
      <div className="grid grid-cols-2 gap-3">
        <Controller
          name="price"
          control={form.control}
          render={({ field }) => (
            <Field data-invalid={!!errors.price}>
              <FieldLabel>Цена, сум</FieldLabel>
              <Input {...field} inputMode="numeric" aria-invalid={!!errors.price} />
              <FieldError errors={[errors.price]} />
            </Field>
          )}
        />
        <Controller
          name="sku"
          control={form.control}
          render={({ field }) => (
            <Field data-invalid={!!errors.sku}>
              <FieldLabel>Артикул (SKU)</FieldLabel>
              <Input {...field} aria-invalid={!!errors.sku} />
              <FieldError errors={[errors.sku]} />
            </Field>
          )}
        />
      </div>
      <Controller
        name="stock"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.stock}>
            <FieldLabel>Остаток</FieldLabel>
            <Input {...field} inputMode="numeric" aria-invalid={!!errors.stock} />
            <FieldError errors={[errors.stock]} />
          </Field>
        )}
      />

      {showApiError && (
        <Alert variant="destructive">
          <AlertDescription>{apiError.message}</AlertDescription>
        </Alert>
      )}

      <div className="flex gap-2">
        <Button type="button" variant="ghost" size="lg" onClick={onDone}>
          Отмена
        </Button>
        <Button size="lg" type="submit" disabled={create.isPending}>
          {create.isPending && <Spinner />}
          Создать
        </Button>
      </div>
    </form>
  );
};
