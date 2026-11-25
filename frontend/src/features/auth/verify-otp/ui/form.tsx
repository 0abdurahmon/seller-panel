import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { useSession } from '@/entities/session';
import {
  getApiErrorMessage,
  getAuth,
  type RequestCodeRequest,
  type VerifyRequest,
} from '@/shared/api';
import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldError,
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  Spinner,
} from '@/shared/ui';
import { verifyOtpSchema, type VerifyOtpValues } from '../model/schema';

const api = getAuth();
const RESEND_COOLDOWN_SECONDS = 60;

interface VerifyOtpFormProps {
  email: string;
}

export const VerifyOtpForm = ({ email }: VerifyOtpFormProps) => {
  const session = useSession();
  const navigate = useNavigate();
  const [cooldown, setCooldown] = useState(RESEND_COOLDOWN_SECONDS);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = window.setInterval(() => {
      setCooldown((prev) => (prev > 1 ? prev - 1 : 0));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [cooldown]);

  const form = useForm<VerifyOtpValues>({
    resolver: zodResolver(verifyOtpSchema),
    defaultValues: { code: '' },
  });

  const verify = useMutation({
    mutationFn: (values: VerifyRequest) => api.postAuthVerify(values),
    onSuccess: ({ token }) => {
      session.start(token);
      navigate('/', { replace: true });
    },
  });

  const resend = useMutation({
    mutationFn: (values: RequestCodeRequest) => api.postAuthRequestCode(values),
    onSuccess: () => {
      setCooldown(RESEND_COOLDOWN_SECONDS);
      verify.reset();
      form.reset();
    },
  });

  const onSubmit = (values: VerifyOtpValues) => {
    verify.mutate({ email, code: values.code });
  };

  const { errors } = form.formState;

  return (
    <form className="flex flex-col gap-6" onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <Controller
        name="code"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.code} className="flex flex-col items-center gap-2">
            <InputOTP
              id="otp-input"
              autoFocus
              maxLength={6}
              value={field.value}
              onChange={(value) => {
                field.onChange(value);
                if (value.length === 6) {
                  verify.mutate({ email, code: value });
                }
              }}
              aria-invalid={!!errors.code}
            >
              <InputOTPGroup>
                <InputOTPSlot index={0} className="size-12 text-lg" />
                <InputOTPSlot index={1} className="size-12 text-lg" />
                <InputOTPSlot index={2} className="size-12 text-lg" />
                <InputOTPSlot index={3} className="size-12 text-lg" />
                <InputOTPSlot index={4} className="size-12 text-lg" />
                <InputOTPSlot index={5} className="size-12 text-lg" />
              </InputOTPGroup>
            </InputOTP>
            <FieldError errors={[errors.code]} />
          </Field>
        )}
      />

      {(verify.isError || resend.isError) && (
        <Alert variant="destructive">
          <AlertDescription>
            {getApiErrorMessage(verify.error ?? resend.error)}
          </AlertDescription>
        </Alert>
      )}

      <Button type="submit" size="lg" className="w-full" disabled={verify.isPending}>
        {verify.isPending && <Spinner />}
        Подтвердить
      </Button>

      <div className="-mt-2 text-center">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={cooldown > 0 || resend.isPending}
          onClick={() => resend.mutate({ email })}
        >
          {resend.isPending && <Spinner />}
          {cooldown > 0 ? `Отправить код повторно (${cooldown} с)` : 'Отправить код повторно'}
        </Button>
      </div>
    </form>
  );
};
