import { useEffect, useRef } from 'react';
import { CheckIcon, Input } from '@/shared/ui';

type SpringState = { x: number; v: number; t: number; k: number; c: number };

const N = 6;
const GAP_PX = 8;
const RADIUS_PX = 14;

function makeSpring(x: number, k: number, c: number, reducedMotion: boolean): SpringState {
  return { x, v: 0, t: x, k, c: reducedMotion ? 2 * Math.sqrt(k) : c };
}

function stepSpring(s: SpringState, dt: number) {
  const a = -s.k * (s.x - s.t) - s.c * s.v;
  s.v += a * dt;
  s.x += s.v * dt;
}

function clamp(v: number, a: number, b: number) {
  return Math.max(a, Math.min(b, v));
}

interface AnimatedOtpProps {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  verified: boolean;
  disabled?: boolean;
  invalid?: boolean;
  autoFocus?: boolean;
}

/**
 * Six separate OTP boxes with spring-animated digit entry.
 * On verification (`verified = true`), the boxes merge into a single green pill with a check mark.
 */
export const AnimatedOtp = ({
  id,
  value,
  onChange,
  verified,
  disabled,
  invalid,
  autoFocus,
}: AnimatedOtpProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const boxRefs = useRef<(HTMLDivElement | null)[]>([]);
  const digitRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const caretRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const okRef = useRef<HTMLDivElement>(null);

  const valueRef = useRef(value);
  const prevCharsRef = useRef<string[]>(Array(N).fill(''));
  const focusedRef = useRef(false);
  const verifiedRef = useRef(verified);
  const invalidRef = useRef(invalid);

  const springsRef = useRef<{
    boxes: { y: SpringState; o: SpringState; a: SpringState }[];
    merge: SpringState;
    green: SpringState;
  } | null>(null);

  useEffect(() => {
    valueRef.current = value;
    verifiedRef.current = verified;
    invalidRef.current = invalid;
  }, [value, verified, invalid]);

  // Trigger digit bounce when value changes
  useEffect(() => {
    const springs = springsRef.current;
    if (!springs) return;

    for (let i = 0; i < N; i++) {
      const ch = value[i] ?? '';
      const prev = prevCharsRef.current[i];
      const digitEl = digitRefs.current[i];
      if (ch && ch !== prev) {
        if (digitEl) digitEl.textContent = ch;
        springs.boxes[i].y.x = -14;
        springs.boxes[i].y.v = 0;
        springs.boxes[i].o.x = 0;
        springs.boxes[i].o.t = 1;
      } else if (!ch) {
        if (digitEl) digitEl.textContent = '';
        springs.boxes[i].o.t = 0;
      }
      prevCharsRef.current[i] = ch;
    }
  }, [value]);

  // Trigger merge -> green pill sequence when verified
  useEffect(() => {
    const springs = springsRef.current;
    if (!springs) return;

    if (!verified) {
      springs.merge.t = 0;
      springs.green.t = 0;
      return;
    }

    const t1 = window.setTimeout(() => {
      springs.merge.t = 1;
    }, 80);
    const t2 = window.setTimeout(() => {
      springs.green.t = 1;
    }, 340);

    return () => {
      window.clearTimeout(t1);
      window.clearTimeout(t2);
    };
  }, [verified]);

  // Main spring physics & imperative render loop
  useEffect(() => {
    if (!springsRef.current) {
      const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      springsRef.current = {
        boxes: Array.from({ length: N }, () => ({
          y: makeSpring(0, 520, 16, reduced),
          o: makeSpring(0, 400, 28, reduced),
          a: makeSpring(0, 300, 26, reduced),
        })),
        merge: makeSpring(0, 260, 24, reduced),
        green: makeSpring(0, 220, 28, reduced),
      };
    }
    const springs = springsRef.current;
    const container = containerRef.current;
    if (!container) return;

    let rafId = 0;
    let last = performance.now();
    let clock = 0;

    const frame = (now: number) => {
      const dt = Math.min((now - last) / 1000, 1 / 30);
      last = now;
      clock += dt;

      const currentVal = valueRef.current;
      const isVerified = verifiedRef.current;
      const isFocused = focusedRef.current;
      const isInvalid = invalidRef.current;

      const sub = 4;
      for (let stepIdx = 0; stepIdx < sub; stepIdx++) {
        const subDt = dt / sub;
        springs.boxes.forEach((b, i) => {
          b.a.t = !isVerified && isFocused && i === Math.min(currentVal.length, N - 1) ? 1 : 0;
          stepSpring(b.y, subDt);
          stepSpring(b.o, subDt);
          stepSpring(b.a, subDt);
        });
        stepSpring(springs.merge, subDt);
        stepSpring(springs.green, subDt);
      }

      const width = container.offsetWidth;
      const bw = Math.min(48, (width - GAP_PX * (N - 1)) / N);
      const m = clamp(springs.merge.x, 0, 1.1);
      const mClamped = clamp(m, 0, 1);
      const gap = GAP_PX * (1 - mClamped);
      const total = bw * N + gap * (N - 1);
      const left0 = (width - total) / 2;
      const g = clamp(springs.green.x, 0, 1);

      springs.boxes.forEach((b, i) => {
        const boxEl = boxRefs.current[i];
        const digitEl = digitRefs.current[i];
        const caretEl = caretRefs.current[i];
        if (!boxEl || !digitEl || !caretEl) return;

        boxEl.style.left = `${(left0 + i * (bw + gap)).toFixed(2)}px`;
        boxEl.style.width = `${(bw + (i < N - 1 ? GAP_PX * mClamped + 1 : 0)).toFixed(2)}px`;

        const innerR = RADIUS_PX * (1 - mClamped);
        boxEl.style.borderRadius =
          i === 0
            ? `${RADIUS_PX}px ${innerR}px ${innerR}px ${RADIUS_PX}px`
            : i === N - 1
              ? `${innerR}px ${RADIUS_PX}px ${RADIUS_PX}px ${innerR}px`
              : `${innerR}px`;

        const a = clamp(b.a.x, 0, 1);
        if (g > 0.01) {
          boxEl.style.backgroundColor = `color-mix(in srgb, var(--success) ${(g * 18).toFixed(1)}%, var(--muted))`;
          boxEl.style.boxShadow = 'none';
        } else if (isInvalid) {
          boxEl.style.backgroundColor = 'var(--muted)';
          boxEl.style.boxShadow = 'inset 0 0 0 1.5px var(--destructive)';
        } else if (a > 0.01) {
          boxEl.style.backgroundColor = `color-mix(in srgb, var(--background) ${(a * 100).toFixed(1)}%, var(--muted))`;
          boxEl.style.boxShadow = `inset 0 0 0 ${(1.5 * a).toFixed(2)}px var(--ring)`;
        } else {
          boxEl.style.backgroundColor = 'var(--muted)';
          boxEl.style.boxShadow = 'none';
        }

        digitEl.style.opacity = (clamp(b.o.x, 0, 1) * (1 - g)).toFixed(3);
        digitEl.style.transform = `translateY(${b.y.x.toFixed(2)}px)`;

        const showCaret = a > 0.5 && !currentVal[i] && (clock * 1.8) % 2 < 1.2;
        caretEl.style.opacity = showCaret ? '1' : '0';
      });

      const okEl = okRef.current;
      if (okEl) {
        okEl.style.opacity = clamp(springs.green.x * 1.6 - 0.5, 0, 1).toFixed(3);
        okEl.style.transform = `translateY(${((1 - g) * 8).toFixed(2)}px)`;
      }

      rafId = requestAnimationFrame(frame);
    };

    rafId = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(rafId);
  }, []);

  return (
    <div ref={containerRef} className="relative mx-auto h-14 w-full max-w-xs select-none">
      {Array.from({ length: N }, (_, i) => (
        <div
          key={i}
          ref={(el) => {
            boxRefs.current[i] = el;
          }}
          className="absolute top-0 grid h-14 place-items-center bg-muted text-2xl font-semibold text-foreground tabular-nums will-change-[transform,width,left]"
        >
          <span
            ref={(el) => {
              digitRefs.current[i] = el;
            }}
            className="block will-change-[transform,opacity]"
          />
          <span
            ref={(el) => {
              caretRefs.current[i] = el;
            }}
            aria-hidden
            className="pointer-events-none absolute top-4 left-1/2 -ml-px h-6 w-0.5 rounded-full bg-foreground opacity-0"
          />
        </div>
      ))}

      <div
        ref={okRef}
        className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center gap-2 text-base font-semibold text-success opacity-0"
      >
        <CheckIcon className="size-5" />
        <span>Подтверждено</span>
      </div>

      <Input
        id={id}
        type="text"
        inputMode="numeric"
        autoComplete="one-time-code"
        maxLength={N}
        autoFocus={autoFocus}
        readOnly={disabled || verified}
        value={value}
        aria-label="6-значный код из Telegram"
        aria-invalid={invalid}
        onFocus={() => {
          focusedRef.current = true;
        }}
        onBlur={() => {
          focusedRef.current = false;
        }}
        onChange={(event) => {
          if (disabled || verified) return;
          const digits = event.target.value.replace(/\D/g, '').slice(0, N);
          onChange(digits);
        }}
        className="absolute inset-0 z-20 size-full cursor-text rounded-2xl border-0 bg-transparent text-transparent caret-transparent opacity-0 shadow-none selection:bg-transparent focus-visible:ring-0 disabled:opacity-0"
      />
    </div>
  );
};
