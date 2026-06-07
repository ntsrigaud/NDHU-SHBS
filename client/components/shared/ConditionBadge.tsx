import { Badge } from '@/components/ui/badge';
import { Condition } from '@/lib/types';
import { cn } from '@/lib/utils';

const styles: Record<Condition, string> = {
  good: 'bg-emerald-100 text-emerald-800 border-emerald-200 hover:bg-emerald-100',
  moderate: 'bg-amber-100 text-amber-800 border-amber-200 hover:bg-amber-100',
  poor: 'bg-rose-100 text-rose-800 border-rose-200 hover:bg-rose-100',
};

const labels: Record<Condition, string> = {
  good: 'Good',
  moderate: 'Moderate',
  poor: 'Poor',
};

export default function ConditionBadge({
  condition,
  className,
}: {
  condition: Condition;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn('border text-[10px] font-semibold capitalize', styles[condition], className)}
    >
      {labels[condition]}
    </Badge>
  );
}
