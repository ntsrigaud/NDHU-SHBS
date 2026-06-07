'use client';

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

const CONDITIONS = [
  { value: 'good', label: 'Good' },
  { value: 'moderate', label: 'Moderate' },
  { value: 'poor', label: 'Poor' },
];

export default function FilterSidebar() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function setParam(key: string, value: string) {
    const params = new URLSearchParams(searchParams.toString());
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key);
    }
    params.delete('page');
    router.push(`${pathname}?${params}`);
  }

  function clearFilters() {
    router.push(pathname);
  }

  const hasFilters = ['condition', 'price_min', 'price_max', 'department'].some((k) =>
    searchParams.get(k),
  );

  return (
    <aside className="space-y-6">
      <div className="space-y-2">
        <Label>Condition</Label>
        <Select
          value={searchParams.get('condition') ?? ''}
          onValueChange={(v: string) => setParam('condition', v === 'all' ? '' : v)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Any condition" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Any condition</SelectItem>
            {CONDITIONS.map((c) => (
              <SelectItem key={c.value} value={c.value}>
                {c.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>Price range (NT$)</Label>
        <div className="flex items-center gap-2">
          <Input
            type="number"
            placeholder="Min"
            min={0}
            defaultValue={searchParams.get('price_min') ?? ''}
            onBlur={(e) => setParam('price_min', e.target.value)}
            className="w-full"
          />
          <span className="text-muted-foreground">–</span>
          <Input
            type="number"
            placeholder="Max"
            min={0}
            defaultValue={searchParams.get('price_max') ?? ''}
            onBlur={(e) => setParam('price_max', e.target.value)}
            className="w-full"
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label>Department</Label>
        <Input
          placeholder="e.g. Computer Science"
          defaultValue={searchParams.get('department') ?? ''}
          onBlur={(e) => setParam('department', e.target.value)}
        />
      </div>

      {hasFilters && (
        <>
          <Separator />
          <Button variant="ghost" className="w-full" onClick={clearFilters}>
            Clear filters
          </Button>
        </>
      )}
    </aside>
  );
}
