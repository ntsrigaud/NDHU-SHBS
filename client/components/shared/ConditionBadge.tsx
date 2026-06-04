import { Badge } from '@/components/ui/badge';
import { Condition } from '@/lib/types';
import { cn } from '@/lib/utils';

const styles: Record<Condition, string> = {
  good: 'bg-green-100 text-green-800 hover:bg-green-100',
  moderate: 'bg-yellow-100 text-yellow-800 hover:bg-yellow-100',
  poor: 'bg-red-100 text-red-800 hover:bg-red-100',
};

const labels: Record<Condition, string> = {
  good: 'Good',
  moderate: 'Moderate',
  poor: 'Poor',
};

export default function ConditionBadge({ condition }: { condition: Condition }) {
  return (
    <Badge variant="secondary" className={cn('capitalize', styles[condition])}>
      {labels[condition]}
    </Badge>
  );
}
