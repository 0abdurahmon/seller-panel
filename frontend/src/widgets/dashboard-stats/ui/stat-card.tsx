import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { cn } from 'cn';
import { Card } from '@/shared/ui';

export type StatTone = 'neutral' | 'success' | 'attention';

const iconTone: Record<StatTone, string> = {
  neutral: 'bg-muted text-muted-foreground',
  success: 'bg-success/10 text-success',
  attention: 'bg-primary text-primary-foreground',
};

/** One number on the dashboard. `attention` marks the one the seller should act on, and `to` makes it a link. */
export const StatCard = ({
  label,
  value,
  unit,
  hint,
  icon,
  tone = 'neutral',
  to,
}: {
  label: string;
  value: string;
  unit?: string;
  hint: ReactNode;
  icon: ReactNode;
  tone?: StatTone;
  to?: string;
}) => {
  const card = (
    <Card
      className={cn(
        'h-full gap-3 px-5 py-5 shadow-none',
        tone === 'attention' && 'bg-primary/5 ring-primary/25',
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        <span
          className={cn(
            'flex size-9 shrink-0 items-center justify-center rounded-xl [&_svg]:size-[18px]',
            iconTone[tone],
          )}
        >
          {icon}
        </span>
      </div>
      <span className="text-3xl font-semibold tracking-tight whitespace-nowrap tabular-nums">
        {value}
        {unit && <span className="ml-1.5 text-base font-medium text-muted-foreground">{unit}</span>}
      </span>
      <span className="text-xs text-muted-foreground">{hint}</span>
    </Card>
  );

  if (!to) return card;
  return (
    <Link
      to={to}
      className="rounded-4xl outline-none transition-transform hover:-translate-y-0.5 focus-visible:ring-3 focus-visible:ring-ring/30"
    >
      {card}
    </Link>
  );
};
