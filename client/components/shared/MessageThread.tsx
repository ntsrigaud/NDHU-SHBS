'use client';

import { useEffect, useRef, useState } from 'react';
import { useMessages, useSendMessage } from '@/hooks/useMessages';
import { useAppSelector } from '@/lib/store';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { Send } from 'lucide-react';

interface Props {
  listingId: string;
  /** Required if the current user is the seller */
  otherUserId?: string;
  /** Name shown for the other party */
  otherUserName: string;
}

export default function MessageThread({ listingId, otherUserId, otherUserName }: Props) {
  const currentUser = useAppSelector((s) => s.auth.user);
  const { data: messages, isLoading } = useMessages(listingId, otherUserId);
  const { mutate: send, isPending } = useSendMessage(listingId, otherUserId);
  const [draft, setDraft] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  function handleSend(e: React.FormEvent) {
    e.preventDefault();
    const body = draft.trim();
    if (!body || isPending) return;
    send(body, { onSuccess: () => setDraft('') });
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b px-4 py-3">
        <p className="font-medium">{otherUserName}</p>
      </div>

      <div className="flex-1 space-y-3 overflow-y-auto p-4">
        {isLoading &&
          Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className={cn('h-10 w-2/3', i % 2 === 0 ? 'ml-auto' : '')} />
          ))}

        {messages?.length === 0 && (
          <p className="py-8 text-center text-sm text-muted-foreground">
            No messages yet. Say hello!
          </p>
        )}

        {messages?.map((msg) => {
          const isMine = msg.sender_id === currentUser?.id;
          return (
            <div key={msg.id} className={cn('flex', isMine ? 'justify-end' : 'justify-start')}>
              <div
                className={cn(
                  'max-w-[75%] rounded-2xl px-4 py-2 text-sm',
                  isMine
                    ? 'rounded-br-sm bg-primary text-primary-foreground'
                    : 'rounded-bl-sm bg-muted'
                )}
              >
                <p>{msg.body}</p>
                <p
                  className={cn(
                    'mt-1 text-right text-[10px]',
                    isMine ? 'text-primary-foreground/70' : 'text-muted-foreground'
                  )}
                >
                  {new Date(msg.created_at).toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </p>
              </div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="flex gap-2 border-t p-4">
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Type a message…"
          maxLength={2000}
          disabled={isPending}
        />
        <Button type="submit" size="icon" disabled={!draft.trim() || isPending}>
          <Send className="h-4 w-4" />
        </Button>
      </form>
    </div>
  );
}
