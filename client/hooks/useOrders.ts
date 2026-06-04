'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';

export interface Order {
  id: string;
  status: string;
  total: number;
  created_at: string;
  items: {
    listing_id: string;
    title: string;
    price: number;
    image_url: string;
  }[];
}

export function useOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: () => apiFetch<Order[]>('/orders'),
  });
}

export function useCreateOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<Order>('/orders', { method: 'POST' }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
  });
}
