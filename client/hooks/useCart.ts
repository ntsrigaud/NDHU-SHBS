'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { cartApi } from '@/lib/api';
import { useAppDispatch } from '@/lib/store';
import { setCart, removeItem } from '@/slices/cartSlice';

export function useCart() {
  const dispatch = useAppDispatch();
  return useQuery({
    queryKey: ['cart'],
    queryFn: async () => {
      const items = await cartApi.getCart();
      dispatch(setCart(items));
      return items;
    },
  });
}

export function useAddToCart() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) => cartApi.addToCart(listingId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cart'] }),
  });
}

export function useRemoveFromCart() {
  const dispatch = useAppDispatch();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) => cartApi.removeFromCart(listingId),
    onSuccess: (_data, listingId) => {
      dispatch(removeItem(listingId));
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
  });
}
