import type { BalanceEntryType } from '@/shared/api';
import { Badge } from '@/shared/ui';

interface EntryTypeBadgeProps {
  type: BalanceEntryType;
}

const LABELS: Record<BalanceEntryType, string> = {
  sale: 'Продажа',
  commission: 'Комиссия',
  payout: 'Выплата',
};

export const EntryTypeBadge = ({ type }: EntryTypeBadgeProps) => {
  if (type === 'sale') {
    return (
      <Badge variant="outline" className="border-success/30 bg-success/10 text-success">
        {LABELS.sale}
      </Badge>
    );
  }

  if (type === 'commission') {
    return (
      <Badge variant="outline" className="border-border bg-muted text-muted-foreground">
        {LABELS.commission}
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="border-primary/30 bg-primary/10 text-primary">
      {LABELS.payout}
    </Badge>
  );
};
