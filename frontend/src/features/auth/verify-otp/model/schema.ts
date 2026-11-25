import z from 'zod';

export const verifyOtpSchema = z.object({
  code: z
    .string()
    .trim()
    .regex(/^\d{6}$/, 'Введите 6 цифр кода'),
});

export type VerifyOtpValues = z.infer<typeof verifyOtpSchema>;
