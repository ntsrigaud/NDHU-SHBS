'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useAppDispatch } from '@/lib/store';
import { setCart, removeItem, CartItem } from '@/slices/cartSlice';

interface ServerCartItem {
  listing_id: string;
  title: string;
  price: number;
  image_url: string;
  seller_name: string;
}

function toCartItem(s: ServerCartItem): CartItem {
  return {
    listingId: s.listing_id,
    title: s.title,
    price: s.price,
    imageUrl: s.image_url,
    sellerName: s.seller_name,
  };
}

export function useCart() {
  const dispatch = useAppDispatch();
  return useQuery({
    queryKey: ['cart'],
    queryFn: async () => {
      const items = await apiFetch<ServerCartItem[]>('/cart');
      dispatch(setCart(items.map(toCartItem)));
      return items;
    },
  });
}

export function useAddToCart() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiFetch<void>('/cart', { method: 'POST', body: JSON.stringify({ listing_id: listingId }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cart'] }),
  });
}

export function useRemoveFromCart() {
  const dispatch = useAppDispatch();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) => apiFetch<void>(`/cart/${listingId}`, { method: 'DELETE' }),
    onSuccess: (_data, listingId) => {
      dispatch(removeItem(listingId));
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
  });
}
