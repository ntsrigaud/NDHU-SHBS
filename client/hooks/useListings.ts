'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { BookListing, Condition } from '@/lib/types';

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
    queryFn: () => apiFetch<BookListing[]>('/listings/me'),
  });
}

export function useCreateListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateListingPayload) =>
      apiFetch<BookListing>('/listings', { method: 'POST', body: JSON.stringify(data) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['my-listings'] }),
  });
}

export function useUpdateListing(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<CreateListingPayload>) =>
      apiFetch<BookListing>(`/listings/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-listings'] });
      qc.invalidateQueries({ queryKey: ['listing', id] });
    },
  });
}

export function useDeleteListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/listings/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['my-listings'] }),
  });
}
