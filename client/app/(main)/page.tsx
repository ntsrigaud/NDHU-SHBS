'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { useBooks } from '@/hooks/useBooks';
import { ListingsFilter } from '@/lib/types';
import BookCard from '@/components/shared/BookCard';
import SearchBar from '@/components/shared/SearchBar';
import FilterSidebar from '@/components/shared/FilterSidebar';
import { Skeleton } from '@/components/ui/skeleton';

function BookGrid() {
  const searchParams = useSearchParams();

  const filters: ListingsFilter = {
    department: searchParams.get('department') ?? undefined,
    condition: (searchParams.get('condition') as ListingsFilter['condition']) ?? undefined,
    price_min: searchParams.get('price_min') ? Number(searchParams.get('price_min')) : undefined,
    price_max: searchParams.get('price_max') ? Number(searchParams.get('price_max')) : undefined,
    sort: (searchParams.get('sort') as ListingsFilter['sort']) ?? 'newest',
    page: searchParams.get('page') ? Number(searchParams.get('page')) : 1,
  };

  const { data, isLoading, isError } = useBooks(filters);

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="aspect-[3/4] w-full rounded-lg" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="py-20 text-center text-muted-foreground">
        Failed to load listings. Please try again later.
      </div>
    );
  }

  if (!data?.listings.length) {
    return (
      <div className="py-20 text-center text-muted-foreground">
        No books found. Try adjusting your filters.
      </div>
    );
  }

  return (
    <div>
      <p className="mb-4 text-sm text-muted-foreground">
        {data.total} book{data.total !== 1 ? 's' : ''} available
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
      <div className="bg-primary px-4 py-10 text-primary-foreground">
        <div className="container mx-auto max-w-2xl text-center">
          <p className="mb-1 text-sm font-medium tracking-widest text-primary-foreground/70 uppercase">
            國立東華大學 · National Dong Hwa University
          </p>
          <h1 className="text-3xl font-bold sm:text-4xl">二手書市集</h1>
          <p className="mt-2 text-primary-foreground/80">
            Buy and sell used textbooks with fellow NDHU students
          </p>
          <div className="mt-6">
            <Suspense>
              <SearchBar />
            </Suspense>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        <div className="flex gap-8">
        <div className="hidden w-56 flex-shrink-0 lg:block">
          <Suspense>
            <FilterSidebar />
          </Suspense>
        </div>

        <div className="min-w-0 flex-1">
          <Suspense
            fallback={
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
                {Array.from({ length: 8 }).map((_, i) => (
                  <Skeleton key={i} className="aspect-[3/4] w-full rounded-lg" />
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
