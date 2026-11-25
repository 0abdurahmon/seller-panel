import z from 'zod';

export const requestCodeSchema = z.object({
  email: z.string().trim().min(1, 'Укажите email').pipe(z.email('Это не похоже на email')),
});

export type RequestCodeValues = z.infer<typeof requestCodeSchema>;
