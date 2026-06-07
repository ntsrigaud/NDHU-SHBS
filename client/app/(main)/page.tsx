'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { useBooks } from '@/hooks/useBooks';
import { ListingsFilter } from '@/lib/types';
import BookCard from '@/components/shared/BookCard';
import SearchBar from '@/components/shared/SearchBar';
import FilterSidebar from '@/components/shared/FilterSidebar';
import { Skeleton } from '@/components/ui/skeleton';
import { BookOpen } from 'lucide-react';

function BookGrid() {
  const searchParams = useSearchParams();

  const filters: ListingsFilter = {
    search: searchParams.get('q') ?? undefined,
    department: searchParams.get('department') ?? undefined,
    condition: (searchParams.get('condition') as ListingsFilter['condition']) ?? undefined,
    price_min: searchParams.get('price_min') ? Number(searchParams.get('price_min')) : undefined,
    price_max: searchParams.get('price_max') ? Number(searchParams.get('price_max')) : undefined,
    page: searchParams.get('page') ? Number(searchParams.get('page')) : 1,
  };

  const { data, isLoading, isError } = useBooks(filters);

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="space-y-3">
            <Skeleton className="aspect-[3/4] w-full rounded-xl" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <div className="mb-4 rounded-full bg-destructive/10 p-4">
          <BookOpen className="h-8 w-8 text-destructive" />
        </div>
        <p className="font-medium text-foreground">Failed to load listings</p>
        <p className="mt-1 text-sm text-muted-foreground">Please try again later.</p>
      </div>
    );
  }

  if (!data?.listings.length) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <div className="mb-4 rounded-full bg-muted p-4">
          <BookOpen className="h-8 w-8 text-muted-foreground" />
        </div>
        <p className="font-medium text-foreground">No books found</p>
        <p className="mt-1 text-sm text-muted-foreground">Try adjusting your search or filters.</p>
      </div>
    );
  }

  return (
    <div>
      <p className="mb-4 text-sm text-muted-foreground">
        <span className="font-semibold text-foreground">{data.total}</span>{' '}
        book{data.total !== 1 ? 's' : ''} available
      </p>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {data.listings.map((listing) => (
          <BookCard key={listing.id} listing={listing} />
        ))}
      </div>
    </div>
  );
}

export default function MarketplacePage() {
  return (
    <div>
      {/* Hero */}
      <div
        className="relative overflow-hidden px-4 py-14"
        style={{ background: 'linear-gradient(135deg, hsl(152 52% 20%) 0%, hsl(165 45% 30%) 60%, hsl(152 40% 38%) 100%)' }}
      >
        {/* decorative dots */}
        <div className="pointer-events-none absolute inset-0 opacity-10"
          style={{ backgroundImage: 'radial-gradient(circle, white 1px, transparent 1px)', backgroundSize: '28px 28px' }}
        />
        <div className="relative container mx-auto max-w-2xl text-center">
          <span className="mb-3 inline-block rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-medium uppercase tracking-widest text-white/80 backdrop-blur-sm">
            國立東華大學 · National Dong Hwa University
          </span>
          <h1 className="mt-2 text-4xl font-extrabold tracking-tight text-white sm:text-5xl">
            二手書市集
          </h1>
          <p className="mt-3 text-base text-white/70">
            Buy and sell used textbooks with fellow NDHU students
          </p>
          <div className="mt-8">
            <Suspense>
              <SearchBar />
            </Suspense>
          </div>
        </div>
      </div>

      {/* Listings */}
      <div className="container mx-auto px-4 py-10">
        <div className="flex gap-8">
          <div className="hidden w-60 flex-shrink-0 lg:block">
            <div className="sticky top-24 rounded-xl border bg-card p-5 shadow-sm">
              <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                Filters
              </h2>
              <Suspense>
                <FilterSidebar />
              </Suspense>
            </div>
          </div>

          <div className="min-w-0 flex-1">
            <Suspense
              fallback={
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
                  {Array.from({ length: 8 }).map((_, i) => (
                    <Skeleton key={i} className="aspect-[3/4] w-full rounded-xl" />
                  ))}
                </div>
              }
            >
              <BookGrid />
            </Suspense>
          </div>
        </div>
      </div>
    </div>
  );
}
