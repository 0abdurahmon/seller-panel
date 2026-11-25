import { useLayoutEffect, useRef, type RefObject } from 'react';

type SpringState = { x: number; v: number; t: number; k: number; c: number };

const LEAD_K = 620;
const LEAD_C = 40;
const TRAIL_K = 170;
const TRAIL_C = 19;

function stepSpring(s: SpringState, dt: number) {
  const a = -s.k * (s.x - s.t) - s.c * s.v;
  s.v += a * dt;
  s.x += s.v * dt;
}

function tuneSpring(s: SpringState, k: number, c: number, reducedMotion: boolean) {
  s.k = k;
  s.c = reducedMotion ? 2 * Math.sqrt(k) : c;
}

/**
 * Two-spring vertical pill that stretches in the direction of travel:
 * the leading edge uses a stiff spring while the trailing edge catches up.
 */
export function useActivePill(
  navRef: RefObject<HTMLElement | null>,
  pillRef: RefObject<HTMLSpanElement | null>,
  activeKey: string,
) {
  const topSpring = useRef<SpringState>({ x: 0, v: 0, t: 0, k: LEAD_K, c: LEAD_C });
  const bottomSpring = useRef<SpringState>({ x: 0, v: 0, t: 0, k: LEAD_K, c: LEAD_C });
  const hasSnapped = useRef(false);
  const prevTop = useRef<number | null>(null);

  useLayoutEffect(() => {
    const nav = navRef.current;
    const pill = pillRef.current;
    if (!nav || !pill) return;

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const A = topSpring.current;
    const B = bottomSpring.current;

    const measure = (snap: boolean) => {
      const active = nav.querySelector<HTMLElement>('[aria-current="page"]');
      if (!active) {
        pill.style.opacity = '0';
        hasSnapped.current = false;
        prevTop.current = null;
        return;
      }

      const navRect = nav.getBoundingClientRect();
      const activeRect = active.getBoundingClientRect();
      const nextTop = activeRect.top - navRect.top;
      const nextBottom = nextTop + activeRect.height;

      pill.style.opacity = '1';
      A.t = nextTop;
      B.t = nextBottom;

      if (snap || !hasSnapped.current || prevTop.current === null) {
        A.x = nextTop;
        A.v = 0;
        B.x = nextBottom;
        B.v = 0;
        hasSnapped.current = true;
      } else {
        const dir = nextTop > prevTop.current ? 1 : -1;
        if (dir > 0) {
          tuneSpring(B, LEAD_K, LEAD_C, reducedMotion);
          tuneSpring(A, TRAIL_K, TRAIL_C, reducedMotion);
        } else {
          tuneSpring(A, LEAD_K, LEAD_C, reducedMotion);
          tuneSpring(B, TRAIL_K, TRAIL_C, reducedMotion);
        }
      }

      prevTop.current = nextTop;
      pill.style.transform = `translateY(${A.x.toFixed(2)}px)`;
      pill.style.height = `${Math.max(0, B.x - A.x).toFixed(2)}px`;
    };

    measure(!hasSnapped.current);

    let rafId = 0;
    let last = performance.now();

    const loop = (now: number) => {
      const dt = Math.min((now - last) / 1000, 1 / 30);
      last = now;

      const settled =
        Math.abs(A.x - A.t) < 0.05 &&
        Math.abs(B.x - B.t) < 0.05 &&
        Math.abs(A.v) < 0.05 &&
        Math.abs(B.v) < 0.05;

      if (!settled) {
        const sub = 4;
        for (let k = 0; k < sub; k++) {
          stepSpring(A, dt / sub);
          stepSpring(B, dt / sub);
        }
        pill.style.transform = `translateY(${A.x.toFixed(2)}px)`;
        pill.style.height = `${Math.max(0, B.x - A.x).toFixed(2)}px`;
        rafId = requestAnimationFrame(loop);
      } else {
        A.x = A.t;
        A.v = 0;
        B.x = B.t;
        B.v = 0;
        pill.style.transform = `translateY(${A.x.toFixed(2)}px)`;
        pill.style.height = `${Math.max(0, B.x - A.x).toFixed(2)}px`;
      }
    };

    rafId = requestAnimationFrame(loop);

    let lastWidth = nav.clientWidth;
    let lastHeight = nav.clientHeight;
    const resizeObserver = new ResizeObserver(() => {
      if (nav.clientWidth === lastWidth && nav.clientHeight === lastHeight) return;
      lastWidth = nav.clientWidth;
      lastHeight = nav.clientHeight;
      measure(true);
    });
    resizeObserver.observe(nav);

    return () => {
      cancelAnimationFrame(rafId);
      resizeObserver.disconnect();
    };
  }, [navRef, pillRef, activeKey]);
}
