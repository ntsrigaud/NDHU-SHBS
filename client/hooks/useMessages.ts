'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { messagesApi, type Conversation, type Message } from '@/lib/api';

export function useMessages(listingId: string) {
  return useQuery({
    queryKey: ['messages', listingId],
    queryFn: () => messagesApi.getMessages(listingId),
    refetchInterval: 5000,
    enabled: !!listingId,
  });
}

export function useSendMessage(listingId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => messagesApi.sendMessage(listingId, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['messages', listingId] }),
  });
}

export function useMarkRead(messageId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => messagesApi.markRead(messageId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['unread-count'] }),
  });
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['unread-count'],
    queryFn: async () => ({ count: await messagesApi.getUnreadCount() }),
    refetchInterval: 10000,
  });
}

export function useConversations() {
  return useQuery({
    queryKey: ['conversations'],
    queryFn: () => messagesApi.getConversations(),
    refetchInterval: 10000,
  });
}
