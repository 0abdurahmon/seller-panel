import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { AnimatedNumber, Card } from '@/shared/ui';

/** One number on the dashboard. All cards look the same; `to` makes one a link. */
export const StatCard = ({
  label,
  value,
  unit,
  hint,
  icon,
  to,
}: {
  label: string;
  value: number;
  unit?: string;
  hint: ReactNode;
  icon: ReactNode;
  to?: string;
}) => {
  const card = (
    <Card className="h-full gap-3 px-5 py-5 shadow-none">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground [&_svg]:size-[18px]">
          {icon}
        </span>
      </div>
      <span className="text-3xl font-semibold tracking-tight whitespace-nowrap">
        <AnimatedNumber value={value} />
        {unit && <span className="ml-1.5 text-base font-medium text-muted-foreground">{unit}</span>}
      </span>
      <span className="text-xs text-muted-foreground">{hint}</span>
    </Card>
  );

  if (!to) return card;
  return (
    <Link to={to} className="rounded-4xl outline-none focus-visible:ring-3 focus-visible:ring-ring/30">
      {card}
    </Link>
  );
};
