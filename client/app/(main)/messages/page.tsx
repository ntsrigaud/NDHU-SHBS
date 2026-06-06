'use client';

import Link from 'next/link';
import { useConversations } from '@/hooks/useMessages';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { MessageSquare } from 'lucide-react';

export default function MessagesPage() {
  const { data: conversations, isLoading } = useConversations();

  return (
    <div className="container mx-auto max-w-2xl px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">Messages</h1>

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      )}

      {conversations?.length === 0 && (
        <div className="py-20 text-center text-muted-foreground">
          <MessageSquare className="mx-auto mb-3 h-10 w-10 opacity-30" />
          <p>No conversations yet.</p>
        </div>
      )}

      {conversations && conversations.length > 0 && (
        <div className="divide-y rounded-lg border">
          {conversations.map((conv) => (
            <Link
              key={conv.listing_id}
              href={`/messages/${conv.listing_id}`}
              className="flex items-start gap-4 p-4 transition-colors hover:bg-muted/50"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className="truncate font-medium">{conv.listing_title}</p>
                  <span className="flex-shrink-0 text-xs text-muted-foreground">
                    {new Date(conv.last_message_at).toLocaleDateString()}
                  </span>
                </div>
                <p className="text-sm text-muted-foreground">{conv.other_user.name}</p>
                <p className="mt-0.5 truncate text-sm text-muted-foreground">{conv.last_message}</p>
              </div>
              {conv.unread_count > 0 && (
                <Badge className="mt-1 flex-shrink-0">{conv.unread_count}</Badge>
              )}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
