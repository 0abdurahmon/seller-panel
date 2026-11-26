import { useCallback, useEffect, useRef, useState } from "react";
import { cn } from "cn";
import { Button } from "./button";
import { MinusIcon, PlusIcon } from "./icons";
import { Input } from "./input";

type Spring = { x: number; v: number; t: number; k: number; c: number };

const createSpring = (x: number, k: number, c: number): Spring => ({ x, v: 0, t: x, k, c });
const clamp = (v: number, a: number, b: number) => Math.max(a, Math.min(b, v));

const stepSpring = (s: Spring, dt: number, reduced: boolean) => {
  const damping = reduced ? 2 * Math.sqrt(s.k) : s.c;
  const a = -s.k * (s.x - s.t) - damping * s.v;
  s.v += a * dt;
  s.x += s.v * dt;
};

const isSettled = (s: Spring) => Math.abs(s.x - s.t) < 0.0005 && Math.abs(s.v) < 0.0005;

export type NumberStepperProps = {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  label?: string;
  description?: string;
  disabled?: boolean;
  "aria-invalid"?: boolean;
  decrementLabel?: string;
  incrementLabel?: string;
  className?: string;
};

/**
 * Spring-animated number stepper adapted from DesignBookmark's Number Stepper
 * to the project's shadcn/Base UI tokens. Supports click-to-type for large numbers.
 */
export function NumberStepper({
  value,
  onChange,
  min = 0,
  max = 1_000_000,
  step = 1,
  label,
  description,
  disabled = false,
  "aria-invalid": ariaInvalid,
  decrementLabel = "Уменьшить",
  incrementLabel = "Увеличить",
  className,
}: NumberStepperProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(String(value));

  const minusRef = useRef<HTMLButtonElement>(null);
  const plusRef = useRef<HTMLButtonElement>(null);
  const numWrapRef = useRef<HTMLDivElement>(null);
  const currRef = useRef<HTMLSpanElement>(null);
  const prevRef = useRef<HTMLSpanElement>(null);

  const prevValueRef = useRef(value);
  const dirRef = useRef(1);
  const rafRef = useRef<number | null>(null);
  const lastTimeRef = useRef<number>(0);

  // Exact stiffness/damping from DesignBookmark number-stepper:
  // roll (260, 24), minus/plus limit kick (600, 14), number nudge (500, 16)
  const springsRef = useRef({
    roll: createSpring(0, 260, 24),
    kickMinus: createSpring(0, 600, 14),
    kickPlus: createSpring(0, 600, 14),
    nudge: createSpring(0, 500, 16),
  });

  const renderFrame = useCallback(() => {
    const { roll, kickMinus, kickPlus, nudge } = springsRef.current;
    const r = roll.x;
    const d = dirRef.current;

    if (currRef.current) {
      currRef.current.style.transform = `translateY(${(d * r * 100).toFixed(2)}%)`;
      currRef.current.style.opacity = clamp(1 - Math.abs(r), 0, 1).toFixed(3);
    }
    if (prevRef.current) {
      prevRef.current.style.transform = `translateY(${((r - 1) * d * 100).toFixed(2)}%)`;
      prevRef.current.style.opacity = clamp(r, 0, 1).toFixed(3);
    }
    if (minusRef.current) {
      minusRef.current.style.transform = `translateX(${(kickMinus.x * 0.06).toFixed(2)}px)`;
    }
    if (plusRef.current) {
      plusRef.current.style.transform = `translateX(${(kickPlus.x * 0.06).toFixed(2)}px)`;
    }
    if (numWrapRef.current) {
      numWrapRef.current.style.transform = `translateX(${(nudge.x * 0.04).toFixed(2)}px)`;
    }
  }, []);

  const startLoop = useCallback(() => {
    if (rafRef.current !== null) return;
    lastTimeRef.current = performance.now();

    const tick = (now: number) => {
      const dt = Math.min((now - lastTimeRef.current) / 1000, 1 / 30);
      lastTimeRef.current = now;
      const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

      const list = Object.values(springsRef.current);
      const sub = 4;
      for (let k = 0; k < sub; k++) {
        for (const s of list) stepSpring(s, dt / sub, reduced);
      }

      renderFrame();

      if (list.every(isSettled)) {
        for (const s of list) {
          s.x = s.t;
          s.v = 0;
        }
        renderFrame();
        rafRef.current = null;
        return;
      }

      rafRef.current = requestAnimationFrame(tick);
    };

    rafRef.current = requestAnimationFrame(tick);
  }, [renderFrame]);

  useEffect(() => {
    return () => {
      if (rafRef.current !== null) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  useEffect(() => {
    const prev = prevValueRef.current;
    if (prev === value) return;

    dirRef.current = value > prev ? 1 : -1;
    if (prevRef.current) prevRef.current.textContent = prev.toLocaleString("ru-RU");
    if (currRef.current) currRef.current.textContent = value.toLocaleString("ru-RU");
    prevValueRef.current = value;

    springsRef.current.roll.x = 1;
    springsRef.current.roll.v = 0;
    startLoop();
  }, [value, startLoop]);

  const handleStep = (delta: number) => {
    if (disabled) return;
    const next = value + delta;
    if (next < min || next > max) {
      const kick = delta < 0 ? springsRef.current.kickMinus : springsRef.current.kickPlus;
      kick.v = 300 * (delta < 0 ? -1 : 1);
      springsRef.current.nudge.v = 160 * delta;
      startLoop();
      return;
    }
    onChange(next);
  };

  const commitDraft = () => {
    const parsed = Number.parseInt(draft.replace(/\s+/g, ""), 10);
    if (!Number.isNaN(parsed)) {
      onChange(clamp(parsed, min, max));
    }
    setEditing(false);
  };

  const atMin = value <= min;
  const atMax = value >= max;

  const controls = (
    <div className={cn("flex items-center gap-1.5", !label && "w-full justify-between")}>
      <Button
        ref={minusRef}
        type="button"
        variant="secondary"
        size="icon-sm"
        disabled={disabled}
        aria-disabled={atMin || undefined}
        aria-label={decrementLabel}
        onClick={() => handleStep(-step)}
        className={cn(
          "rounded-full will-change-transform transition-opacity",
          atMin && "opacity-40",
        )}
      >
        <MinusIcon className="size-4" />
      </Button>

      <div
        ref={numWrapRef}
        className={cn(
          "relative h-8 overflow-hidden tabular-nums text-foreground will-change-transform",
          label ? "min-w-12 px-1" : "flex-1",
        )}
      >
        {editing ? (
          <Input
            type="text"
            inputMode="numeric"
            value={draft}
            autoFocus
            onFocus={(e) => e.currentTarget.select()}
            onChange={(e) => setDraft(e.target.value.replace(/[^\d]/g, ""))}
            onBlur={commitDraft}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                commitDraft();
              } else if (e.key === "Escape") {
                setEditing(false);
              }
            }}
            className="h-8 rounded-full bg-transparent px-2 text-center text-base tabular-nums focus-visible:ring-0 md:text-sm"
          />
        ) : (
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              setDraft(String(value));
              setEditing(true);
            }}
            aria-live="polite"
            title="Нажмите, чтобы ввести число"
            className="relative grid size-full cursor-text place-items-center rounded-full text-base outline-none focus-visible:ring-2 focus-visible:ring-ring/40 md:text-sm"
          >
            <span
              ref={currRef}
              className="absolute inset-0 grid place-items-center will-change-[transform,opacity]"
            >
              {value.toLocaleString("ru-RU")}
            </span>
            <span
              ref={prevRef}
              aria-hidden
              className="pointer-events-none absolute inset-0 grid place-items-center opacity-0 will-change-[transform,opacity]"
            />
          </button>
        )}
      </div>

      <Button
        ref={plusRef}
        type="button"
        variant="secondary"
        size="icon-sm"
        disabled={disabled}
        aria-disabled={atMax || undefined}
        aria-label={incrementLabel}
        onClick={() => handleStep(step)}
        className={cn(
          "rounded-full will-change-transform transition-opacity",
          atMax && "opacity-40",
        )}
      >
        <PlusIcon className="size-4" />
      </Button>
    </div>
  );

  return (
    <div
      data-slot="number-stepper"
      aria-invalid={ariaInvalid}
      className={cn(
        "flex items-center justify-between rounded-3xl border border-transparent bg-input/50 transition-[color,box-shadow,background-color] aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40",
        label ? "px-4 py-2.5" : "h-11 px-1.5",
        disabled && "pointer-events-none opacity-50",
        className,
      )}
    >
      {label && (
        <div className="min-w-0 pr-3">
          <div className="text-sm font-medium text-foreground">{label}</div>
          {description && (
            <div className="truncate text-xs text-muted-foreground">{description}</div>
          )}
        </div>
      )}
      {controls}
    </div>
  );
}
