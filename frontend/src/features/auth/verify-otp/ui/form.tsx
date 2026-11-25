import { useEffect, useRef, useState } from 'react';
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
  Spinner,
} from '@/shared/ui';
import { verifyOtpSchema, type VerifyOtpValues } from '../model/schema';
import { AnimatedOtp } from './animated-otp';

const api = getAuth();
const RESEND_COOLDOWN_SECONDS = 60;
const VERIFIED_REDIRECT_DELAY_MS = 1200;

interface VerifyOtpFormProps {
  email: string;
}

export const VerifyOtpForm = ({ email }: VerifyOtpFormProps) => {
  const session = useSession();
  const navigate = useNavigate();
  const [cooldown, setCooldown] = useState(RESEND_COOLDOWN_SECONDS);
  const [verified, setVerified] = useState(false);
  const redirectTimerRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (redirectTimerRef.current !== null) {
        window.clearTimeout(redirectTimerRef.current);
      }
    };
  }, []);

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
      setVerified(true);
      const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      redirectTimerRef.current = window.setTimeout(
        () => {
          session.start(token);
          navigate('/', { replace: true });
        },
        reducedMotion ? 350 : VERIFIED_REDIRECT_DELAY_MS,
      );
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
    if (verified) return;
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
            <AnimatedOtp
              id="otp-input"
              autoFocus
              value={field.value}
              verified={verified}
              disabled={verify.isPending}
              invalid={!!errors.code}
              onChange={(digits) => {
                field.onChange(digits);
                if (digits.length === 6 && !verify.isPending && !verified) {
                  verify.mutate({ email, code: digits });
                }
              }}
            />
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

      {verify.isPending && (
        <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
          <Spinner />
          <span>Проверяем код…</span>
        </div>
      )}

      <div className="text-center">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={cooldown > 0 || resend.isPending || verified}
          onClick={() => resend.mutate({ email })}
        >
          {resend.isPending && <Spinner data-icon="inline-start" />}
          {cooldown > 0 ? `Отправить код повторно (${cooldown} с)` : 'Отправить код повторно'}
        </Button>
      </div>
    </form>
  );
};
