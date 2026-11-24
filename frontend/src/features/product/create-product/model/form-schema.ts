import z from "zod";

// Same limits as CreateProductRequest in api/openapi.yaml. The backend checks them again.
const wholeNumber = (label: string, min: number, max: number) =>
  z
    .string()
    .trim()
    .min(1, `Укажите ${label}`)
    .regex(/^\d+$/, "Только целое число")
    .transform(Number)
    .refine((n) => n >= min && n <= max, `От ${min} до ${max.toLocaleString("ru-RU")}`);

export const formSchema = z.object({
  category_id: z.string().min(1, "Выберите категорию").transform(Number),
  title: z.string().trim().min(1, "Укажите название").max(200, "Не длиннее 200 символов"),
  description: z.string().trim().max(5000, "Не длиннее 5000 символов"),
  price: wholeNumber("цену", 1, 1_000_000_000_000),
  sku: z.string().trim().min(1, "Укажите артикул").max(64, "Не длиннее 64 символов"),
  stock: wholeNumber("остаток", 0, 1_000_000),
});

export type FormInput = z.input<typeof formSchema>;
export type FormOutput = z.output<typeof formSchema>;
