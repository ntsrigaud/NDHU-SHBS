'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';

export interface Message {
  id: string;
  listing_id: string;
  sender_id: string;
  receiver_id: string;
  body: string;
  is_read: boolean;
  created_at: string;
  sender?: { id: string; name: string };
}

export interface Conversation {
  listing_id: string;
  listing_title: string;
  other_user: { id: string; name: string };
  last_message: string;
  last_message_at: string;
  unread_count: number;
}

export function useMessages(listingId: string) {
  return useQuery({
    queryKey: ['messages', listingId],
    queryFn: () => apiFetch<Message[]>(`/listings/${listingId}/messages`),
    refetchInterval: 5000,
    enabled: !!listingId,
  });
}

export function useSendMessage(listingId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) =>
      apiFetch<Message>(`/listings/${listingId}/messages`, {
        method: 'POST',
        body: JSON.stringify({ body }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['messages', listingId] }),
  });
}

export function useMarkRead(messageId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<void>(`/messages/${messageId}/read`, { method: 'PATCH' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['unread-count'] }),
  });
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['unread-count'],
    queryFn: () => apiFetch<{ count: number }>('/messages/unread-count'),
    refetchInterval: 10000,
  });
}

export function useConversations() {
  return useQuery({
    queryKey: ['conversations'],
    queryFn: () => apiFetch<Conversation[]>('/messages/conversations'),
    refetchInterval: 10000,
  });
}
