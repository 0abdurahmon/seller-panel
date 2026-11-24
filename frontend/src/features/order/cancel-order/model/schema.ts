import z from "zod";

// The backend requires a non-empty reason to cancel and checks it again.
export const cancelOrderSchema = z.object({
  reason: z.string().trim().min(1, "Укажите причину отмены"),
});

export type CancelOrderValues = z.infer<typeof cancelOrderSchema>;
