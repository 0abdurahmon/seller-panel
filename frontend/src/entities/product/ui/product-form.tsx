import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldDescription,
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
import { PhotosField } from "./photos-field";

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
  const errorOf = (name: FieldName): { message: string } | undefined => {
    // For the photos array, zod's array-level message sits on the field itself or on `root`.
    const error = errors[name] as { message?: string; root?: { message?: string } } | undefined;
    const message = error?.message ?? error?.root?.message ?? fieldErrors?.[name];
    return message ? { message } : undefined;
  };

  const placeholder =
    categoriesStatus === "pending"
      ? "Загрузка категорий…"
      : categoriesStatus === "error"
        ? "Не удалось загрузить категории"
        : "Выберите категорию";

  return (
    <form className="space-y-3" onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <Controller
        name="photos"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errorOf("photos")}>
            <FieldLabel htmlFor="product-photos">Фото</FieldLabel>
            <PhotosField
              id="product-photos"
              value={field.value}
              onChange={field.onChange}
              invalid={!!errorOf("photos")}
            />
            {errorOf("photos") ? (
              <FieldError errors={[errorOf("photos")]} />
            ) : (
              <FieldDescription>
                До 5 фото: JPEG, PNG или WebP, не больше 5 МБ. Первое станет обложкой.
              </FieldDescription>
            )}
          </Field>
        )}
      />
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
