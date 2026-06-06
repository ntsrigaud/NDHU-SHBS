'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ordersApi, type Order } from '@/lib/api';

export function useOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: () => ordersApi.getOrders(),
  });
}

export function useCreateOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => ordersApi.checkout(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
  });
}
