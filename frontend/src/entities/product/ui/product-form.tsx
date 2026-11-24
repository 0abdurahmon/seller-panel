import { Controller, useForm, type FieldError as FormFieldError } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
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
import {
  productFormSchema,
  type ProductFormInput,
  type ProductFormValues,
} from "../model/form-schema";

type FieldName = keyof ProductFormInput;

export type CategoryOption = { value: string; label: string };

export type ProductFormProps = {
  defaultValues: ProductFormInput;
  categories: CategoryOption[];
  categoriesStatus: "pending" | "error" | "success";
  /** Stock is set on create only; later it changes through its own endpoint. */
  withStock: boolean;
  submitLabel: string;
  pending: boolean;
  /** An API error that is not about one field. */
  errorMessage?: string | null;
  /** API errors about one field, for example a taken SKU. */
  fieldErrors?: Partial<Record<FieldName, string>>;
  onSubmit: (values: ProductFormValues) => void;
  onCancel: () => void;
};

export const ProductForm = ({
  defaultValues,
  categories,
  categoriesStatus,
  withStock,
  submitLabel,
  pending,
  errorMessage,
  fieldErrors,
  onSubmit,
  onCancel,
}: ProductFormProps) => {
  const form = useForm<ProductFormInput, unknown, ProductFormValues>({
    resolver: zodResolver(productFormSchema),
    defaultValues,
  });

  const { errors } = form.formState;
  // A client error wins; an API error shows until the form is submitted again.
  const errorOf = (name: FieldName): FormFieldError | undefined =>
    errors[name] ?? (fieldErrors?.[name] ? { type: "server", message: fieldErrors[name] } : undefined);

  const placeholder =
    categoriesStatus === "pending"
      ? "Загрузка категорий…"
      : categoriesStatus === "error"
        ? "Не удалось загрузить категории"
        : "Выберите категорию";

  return (
    <form className="space-y-3" onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <Controller
        name="category_id"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errorOf("category_id")}>
            <FieldLabel>Категория</FieldLabel>
            <Select
              items={categories}
              value={field.value || null}
              onValueChange={(value) => field.onChange(value ?? "")}
              disabled={categoriesStatus !== "success"}
            >
              <SelectTrigger
                className="w-full"
                aria-invalid={!!errorOf("category_id")}
                onBlur={field.onBlur}
              >
                <SelectValue placeholder={placeholder} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {categories.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldError errors={[errorOf("category_id")]} />
          </Field>
        )}
      />
      <Controller
        name="title"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errorOf("title")}>
            <FieldLabel>Название</FieldLabel>
            <Input {...field} aria-invalid={!!errorOf("title")} />
            <FieldError errors={[errorOf("title")]} />
          </Field>
        )}
      />
      <Controller
        name="description"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errorOf("description")}>
            <FieldLabel>Описание</FieldLabel>
            <Textarea {...field} aria-invalid={!!errorOf("description")} />
            <FieldError errors={[errorOf("description")]} />
          </Field>
        )}
      />
      <div className="grid grid-cols-2 gap-3">
        <Controller
          name="price"
          control={form.control}
          render={({ field }) => (
            <Field data-invalid={!!errorOf("price")}>
              <FieldLabel>Цена, сум</FieldLabel>
              <Input {...field} inputMode="numeric" aria-invalid={!!errorOf("price")} />
              <FieldError errors={[errorOf("price")]} />
            </Field>
          )}
        />
        <Controller
          name="sku"
          control={form.control}
          render={({ field }) => (
            <Field data-invalid={!!errorOf("sku")}>
              <FieldLabel>Артикул (SKU)</FieldLabel>
              <Input {...field} aria-invalid={!!errorOf("sku")} />
              <FieldError errors={[errorOf("sku")]} />
            </Field>
          )}
        />
      </div>
      {withStock && (
        <Controller
          name="stock"
          control={form.control}
          render={({ field }) => (
            <Field data-invalid={!!errorOf("stock")}>
              <FieldLabel>Остаток</FieldLabel>
              <Input {...field} inputMode="numeric" aria-invalid={!!errorOf("stock")} />
              <FieldError errors={[errorOf("stock")]} />
            </Field>
          )}
        />
      )}

      {errorMessage && (
        <Alert variant="destructive">
          <AlertDescription>{errorMessage}</AlertDescription>
        </Alert>
      )}

      <div className="flex gap-2">
        <Button type="button" variant="ghost" size="lg" onClick={onCancel}>
          Отмена
        </Button>
        <Button size="lg" type="submit" disabled={pending}>
          {pending && <Spinner />}
          {submitLabel}
        </Button>
      </div>
    </form>
  );
};
