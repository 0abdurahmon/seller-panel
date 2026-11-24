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

/** Same limits as the backend's photo store: up to 5 images, jpeg, png or webp, 5 MB each. */
export const PHOTO_LIMITS = {
  max: 5,
  maxBytes: 5 * 1024 * 1024,
  types: ["image/jpeg", "image/png", "image/webp"],
} as const;

/** A photo in the form: one the product already has (`url`) or a new file to upload (`file`). */
export type PhotoItem = { id: string; url: string; file?: never } | { id: string; file: File; url?: never };

export const photosSchema = z
  .array(z.custom<PhotoItem>())
  // MVP item 3: a product has at least one photo.
  .min(1, "Добавьте хотя бы одно фото")
  .max(PHOTO_LIMITS.max, `Не больше ${PHOTO_LIMITS.max} фото`)
  .refine(
    (items) => items.every((p) => !p.file || (PHOTO_LIMITS.types as readonly string[]).includes(p.file.type)),
    "Только JPEG, PNG или WebP",
  )
  .refine(
    (items) => items.every((p) => !p.file || p.file.size <= PHOTO_LIMITS.maxBytes),
    "Каждое фото не больше 5 МБ",
  );

export const productFormSchema = z.object({
  photos: photosSchema,
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
  photos: [],
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
    photos: product.photos.map((url) => ({ id: url, url })),
    category_id: String(product.category_id),
    title: product.title,
    description: product.description,
    price: String(product.price),
    sku: product.sku,
    stock: String(product.stock),
  };
}
