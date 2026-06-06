'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { listingsApi } from '@/lib/api';
import { Condition } from '@/lib/types';

export interface CreateListingPayload {
  title: string;
  author: string;
  isbn?: string;
  course_code?: string;
  department?: string;
  price: number;
  condition: Condition;
  description?: string;
  image_ids?: string[];
}

export function useMyListings() {
  return useQuery({
    queryKey: ['my-listings'],
    queryFn: () => listingsApi.getMyListings(),
  });
}

export function useCreateListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateListingPayload) => listingsApi.createListing(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['my-listings'] }),
  });
}

export function useUpdateListing(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<CreateListingPayload>) => listingsApi.updateListing(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-listings'] });
      qc.invalidateQueries({ queryKey: ['listing', id] });
    },
  });
}

export function useDeleteListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => listingsApi.deleteListing(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['my-listings'] }),
  });
}
