'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { Search } from 'lucide-react';

export default function SearchBar() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [value, setValue] = useState(searchParams.get('q') ?? '');

  useEffect(() => {
    const timer = setTimeout(() => {
      const params = new URLSearchParams(searchParams.toString());
      if (value) {
        params.set('q', value);
      } else {
        params.delete('q');
      }
      params.delete('page');
      router.push(`${pathname}?${params}`);
    }, 350);
    return () => clearTimeout(timer);
  }, [value, pathname, router, searchParams]);

  return (
    <div className="relative">
      <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/50" />
      <input
        type="search"
        className="h-12 w-full rounded-xl border border-white/20 bg-white/15 pl-11 pr-4 text-sm text-white placeholder:text-white/50 backdrop-blur-sm transition-colors focus:border-white/40 focus:bg-white/20 focus:outline-none focus:ring-0"
        placeholder="Search by title, author, or ISBN…"
        value={value}
        onChange={(e) => setValue(e.target.value)}
      />
    </div>
  );
}
