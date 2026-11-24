import z from "zod";
import type { Product } from "./types";

// Same limits as Create/UpdateProductRequest in api/openapi.yaml. The backend checks them again.
const wholeNumber = (label: string, min: number, max: number) =>
  z
    .string()
    .trim()
    .min(1, `Укажите ${label}`)
    .regex(/^\d+$/, "Только целое число")
    .transform(Number)
    .refine((n) => n >= min && n <= max, `От ${min} до ${max.toLocaleString("ru-RU")}`);

export const productFormSchema = z.object({
  category_id: z.string().min(1, "Выберите категорию").transform(Number),
  title: z.string().trim().min(1, "Укажите название").max(200, "Не длиннее 200 символов"),
  description: z.string().trim().max(5000, "Не длиннее 5000 символов"),
  price: wholeNumber("цену", 1, 1_000_000_000_000),
  sku: z.string().trim().min(1, "Укажите артикул").max(64, "Не длиннее 64 символов"),
  stock: wholeNumber("остаток", 0, 1_000_000),
});

export type ProductFormInput = z.input<typeof productFormSchema>;
export type ProductFormValues = z.output<typeof productFormSchema>;

export const emptyProductForm: ProductFormInput = {
  category_id: "",
  title: "",
  description: "",
  price: "",
  sku: "",
  stock: "0",
};

/** Form values for editing an existing product. */
export function productToForm(product: Product): ProductFormInput {
  return {
    category_id: String(product.category_id),
    title: product.title,
    description: product.description,
    price: String(product.price),
    sku: product.sku,
    stock: String(product.stock),
  };
}
