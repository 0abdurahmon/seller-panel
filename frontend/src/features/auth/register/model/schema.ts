import z from "zod";

// Same limits as RegisterRequest in api/openapi.yaml. The backend checks them again.
export const registerSchema = z.object({
  shop_name: z
    .string()
    .trim()
    .min(1, "Укажите название магазина")
    .max(120, "Не длиннее 120 символов"),
  email: z
    .string()
    .trim()
    .min(1, "Укажите email")
    .max(254, "Слишком длинный email")
    .pipe(z.email("Это не похоже на email")),
  password: z
    .string()
    .min(8, "Не короче 8 символов")
    // bcrypt reads only 72 bytes, so the backend counts bytes, not letters.
    .refine((value) => new TextEncoder().encode(value).length <= 72, "Слишком длинный пароль"),
});

export type RegisterValues = z.infer<typeof registerSchema>;
