'use client';

import { useQuery } from '@tanstack/react-query';
import type { ListingsFilter } from '@/lib/types';
import { listingsApi } from '@/lib/api';

export function useBooks(filters: ListingsFilter = {}) {
  return useQuery({
    queryKey: ['listings', filters],
    queryFn: () => listingsApi.getListings(filters),
  });
}

export function useBook(id: string) {
  return useQuery({
    queryKey: ['listing', id],
    queryFn: () => listingsApi.getListing(id),
    enabled: !!id,
  });
}
