import { useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  shopFormSchema,
  shopKeys,
  shopToForm,
  type Shop,
  type ShopFormInput,
  type ShopFormValues,
} from '@/entities/shop';
import { getShop, multipart, parseApiError } from '@/shared/api';
import {
  Alert,
  AlertDescription,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  CheckIcon,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  PhoneInput,
  Spinner,
  Textarea,
} from '@/shared/ui';
import { LogoField } from './logo-field';

const api = getShop();

type FieldName = keyof ShopFormInput;

export const EditShopForm = ({ shop }: { shop: Shop }) => {
  const queryClient = useQueryClient();
  const [saved, setSaved] = useState(false);

  const form = useForm<ShopFormInput, unknown, ShopFormValues>({
    resolver: zodResolver(shopFormSchema),
    defaultValues: shopToForm(shop),
  });
  const watchedName = useWatch({ control: form.control, name: 'name' });

  const update = useMutation({
    mutationFn: ({ logo, ...fields }: ShopFormValues) => {
      if (logo) {
        return api.putShop({ ...fields, logo }, multipart);
      }
      return api.putShop(fields);
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(shopKeys.current, updated);
      void queryClient.invalidateQueries({ queryKey: shopKeys.current });
      form.reset(shopToForm(updated));
      setSaved(true);
    },
  });

  const { errors, isDirty } = form.formState;
  const apiError = update.error ? parseApiError(update.error) : null;

  const errorOf = (name: FieldName): { message: string } | undefined => {
    const message = errors[name]?.message ?? apiError?.fields[name];
    return message ? { message } : undefined;
  };

  const hasFieldErrors = Object.keys(apiError?.fields ?? {}).length > 0;
  const generalError = apiError && !hasFieldErrors ? apiError.message : null;

  return (
    <form
      className="mx-auto flex w-full max-w-2xl flex-col gap-6"
      onSubmit={form.handleSubmit((values) => {
        setSaved(false);
        update.mutate(values);
      })}
      onChange={() => {
        if (saved) setSaved(false);
      }}
      noValidate
    >
      <Controller
        name="logo"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errorOf('logo')} className="items-center text-center">
            <FieldLabel htmlFor="shop-logo" className="sr-only">
              Логотип
            </FieldLabel>
            <LogoField
              id="shop-logo"
              shop={{ name: watchedName || shop.name, logo_url: shop.logo_url }}
              file={field.value}
              onChange={(file) => {
                setSaved(false);
                field.onChange(file);
              }}
              invalid={!!errorOf('logo')}
            />
            <FieldError errors={[errorOf('logo')]} />
          </Field>
        )}
      />

      <Card className="w-full shadow-none">
        <CardHeader>
          <CardTitle>Профиль магазина</CardTitle>
          <CardDescription>
            Эти данные видят покупатели на витрине и в информации о заказе.
          </CardDescription>
        </CardHeader>

        <CardContent>
          <FieldGroup>

            <Controller
              name="name"
              control={form.control}
              render={({ field }) => (
                <Field data-invalid={!!errorOf('name')}>
                  <FieldLabel htmlFor="shop-name">Название магазина</FieldLabel>
                  <Input
                    {...field}
                    id="shop-name"
                    placeholder="Например, Dona Electronics"
                    maxLength={120}
                    aria-invalid={!!errorOf('name')}
                  />
                  <FieldError errors={[errorOf('name')]} />
                </Field>
              )}
            />

            <Controller
              name="description"
              control={form.control}
              render={({ field }) => (
                <Field data-invalid={!!errorOf('description')}>
                  <FieldLabel htmlFor="shop-description">Описание</FieldLabel>
                  <Textarea
                    {...field}
                    id="shop-description"
                    rows={4}
                    placeholder="Коротко расскажите о магазине и ассортименте"
                    maxLength={2000}
                    aria-invalid={!!errorOf('description')}
                  />
                  <FieldError errors={[errorOf('description')]} />
                </Field>
              )}
            />

            <div className="grid gap-4 sm:grid-cols-2">
              <Controller
                name="contact_phone"
                control={form.control}
                render={({ field }) => (
                  <Field data-invalid={!!errorOf('contact_phone')}>
                    <FieldLabel htmlFor="shop-phone">Телефон для связи</FieldLabel>
                    <PhoneInput
                      {...field}
                      id="shop-phone"
                      aria-invalid={!!errorOf('contact_phone')}
                    />
                    <FieldError errors={[errorOf('contact_phone')]} />
                  </Field>
                )}
              />

              <Controller
                name="pickup_address"
                control={form.control}
                render={({ field }) => (
                  <Field data-invalid={!!errorOf('pickup_address')}>
                    <FieldLabel htmlFor="shop-address">Адрес самовывоза</FieldLabel>
                    <Input
                      {...field}
                      id="shop-address"
                      placeholder="г. Ташкент, ул. Амира Темура, 1"
                      maxLength={300}
                      aria-invalid={!!errorOf('pickup_address')}
                    />
                    <FieldError errors={[errorOf('pickup_address')]} />
                  </Field>
                )}
              />
            </div>

            {generalError && (
              <Alert variant="destructive">
                <AlertDescription>{generalError}</AlertDescription>
              </Alert>
            )}

            {saved && !isDirty && (
              <Alert className="border-success/30 bg-success/10 text-success">
                <CheckIcon />
                <AlertDescription className="text-success">
                  Изменения сохранены
                </AlertDescription>
              </Alert>
            )}
          </FieldGroup>
        </CardContent>

        <CardFooter className="mt-6 justify-end gap-2">
          {isDirty && (
            <Button
              type="button"
              variant="ghost"
              disabled={update.isPending}
              onClick={() => {
                form.reset(shopToForm(shop));
                update.reset();
              }}
            >
              Отменить
            </Button>
          )}
          <Button type="submit" disabled={update.isPending || !isDirty}>
            {update.isPending && <Spinner data-icon="inline-start" />}
            Сохранить
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
