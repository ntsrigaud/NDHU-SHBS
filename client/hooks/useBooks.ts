'use client';

import { useQuery } from '@tanstack/react-query';
import { BookListing, ListingsFilter } from '@/lib/types';

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

async function fetchListings(
  filters: ListingsFilter,
): Promise<{ listings: BookListing[]; total: number }> {
  const params = new URLSearchParams();
  params.set('status', 'active');
  if (filters.department) params.set('department', filters.department);
  if (filters.condition) params.set('condition', filters.condition);
  if (filters.price_min != null) params.set('price_min', String(filters.price_min));
  if (filters.price_max != null) params.set('price_max', String(filters.price_max));
  if (filters.sort) params.set('sort', filters.sort);
  params.set('page', String(filters.page ?? 1));
  params.set('limit', String(filters.limit ?? 20));

  const res = await fetch(`${BASE_URL}/api/v1/listings?${params}`, {
    credentials: 'include',
  });
  if (!res.ok) throw new Error('Failed to fetch listings');

  const total = Number(res.headers.get('X-Total-Count') ?? 0);
  const listings: BookListing[] = await res.json();
  return { listings, total };
}

async function fetchListing(id: string): Promise<BookListing> {
  const res = await fetch(`${BASE_URL}/api/v1/listings/${id}`, {
    credentials: 'include',
  });
  if (!res.ok) throw new Error('Listing not found');
  return res.json();
}

export function useBooks(filters: ListingsFilter = {}) {
  return useQuery({
    queryKey: ['listings', filters],
    queryFn: () => fetchListings(filters),
  });
}

export function useBook(id: string) {
  return useQuery({
    queryKey: ['listing', id],
    queryFn: () => fetchListing(id),
    enabled: !!id,
  });
}
