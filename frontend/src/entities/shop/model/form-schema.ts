import z from 'zod';
import type { Shop } from '@/shared/api';
import { formatPhoneNumber } from '@/shared/lib';

/** Same limits as the backend's image store: jpeg, png or webp, up to 5 MB. */
export const LOGO_LIMITS = {
  maxBytes: 5 * 1024 * 1024,
  types: ['image/jpeg', 'image/png', 'image/webp'],
} as const;

export const shopFormSchema = z.object({
  logo: z
    .custom<File | null>()
    .nullable()
    .refine(
      (file) => !file || (LOGO_LIMITS.types as readonly string[]).includes(file.type),
      'Только JPEG, PNG или WebP',
    )
    .refine((file) => !file || file.size <= LOGO_LIMITS.maxBytes, 'Логотип не больше 5 МБ'),
  name: z.string().trim().min(1, 'Укажите название магазина').max(120, 'Не длиннее 120 символов'),
  description: z.string().trim().max(2000, 'Не длиннее 2000 символов'),
  contact_phone: z
    .string()
    .trim()
    .transform((value) => {
      const normalized = value.replace(/[\s\-()]/g, '');
      return normalized === '+998' ? '' : value;
    })
    .refine((value) => {
      if (!value) return true;
      const normalized = value.replace(/[\s\-()]/g, '');
      return /^\+?\d{7,15}$/.test(normalized);
    }, 'Укажите от 7 до 15 цифр (можно с +)'),
  pickup_address: z.string().trim().max(300, 'Не длиннее 300 символов'),
});

export type ShopFormInput = z.input<typeof shopFormSchema>;
export type ShopFormValues = z.output<typeof shopFormSchema>;

export function shopToForm(shop: Shop): ShopFormInput {
  return {
    logo: null,
    name: shop.name,
    description: shop.description ?? '',
    contact_phone: formatPhoneNumber(shop.contact_phone ?? ''),
    pickup_address: shop.pickup_address ?? '',
  };
}
