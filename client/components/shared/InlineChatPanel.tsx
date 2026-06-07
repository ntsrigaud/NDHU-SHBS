'use client';

import { useEffect, useRef, useState } from 'react';
import { useMessages, useSendMessage } from '@/hooks/useMessages';
import { useAppSelector } from '@/lib/store';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { Send, ChevronDown, MessageSquare } from 'lucide-react';

interface Props {
  listingId: string;
  sellerName: string;
}

export default function InlineChatPanel({ listingId, sellerName }: Props) {
  const currentUser = useAppSelector((s) => s.auth.user);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  const { data: messages, isLoading } = useMessages(listingId);
  const { mutate: send, isPending } = useSendMessage(listingId);

  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, open]);

  function handleSend(e: React.FormEvent) {
    e.preventDefault();
    const body = draft.trim();
    if (!body || isPending) return;
    send(body, { onSuccess: () => setDraft('') });
  }

  return (
    <div className="overflow-hidden rounded-xl border bg-card shadow-sm">
      {/* Header — always visible */}
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-muted/50"
      >
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary">
            <MessageSquare className="h-4 w-4" />
          </div>
          <div>
            <p className="text-sm font-semibold">Message {sellerName}</p>
            <p className="text-xs text-muted-foreground">Ask about condition, availability, or price</p>
          </div>
        </div>
        <ChevronDown className={cn('h-4 w-4 text-muted-foreground transition-transform duration-200', open && 'rotate-180')} />
      </button>

      {/* Expandable thread */}
      {open && (
        <div className="border-t">
          {/* Messages */}
          <div className="flex h-64 flex-col gap-3 overflow-y-auto p-4">
            {isLoading && Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className={cn('h-9 w-2/3 rounded-2xl', i % 2 !== 0 && 'ml-auto')} />
            ))}

            {!isLoading && messages?.length === 0 && (
              <div className="flex flex-1 flex-col items-center justify-center text-center">
                <MessageSquare className="mb-2 h-8 w-8 text-muted-foreground/40" />
                <p className="text-sm text-muted-foreground">No messages yet</p>
                <p className="text-xs text-muted-foreground">Start the conversation below</p>
              </div>
            )}

            {messages?.map((msg) => {
              const isMine = msg.sender_id === currentUser?.id;
              return (
                <div key={msg.id} className={cn('flex', isMine ? 'justify-end' : 'justify-start')}>
                  <div className={cn(
                    'max-w-[80%] rounded-2xl px-3.5 py-2 text-sm',
                    isMine ? 'rounded-br-sm bg-primary text-primary-foreground' : 'rounded-bl-sm bg-muted',
                  )}>
                    <p>{msg.body}</p>
                    <p className={cn('mt-0.5 text-right text-[10px]', isMine ? 'text-primary-foreground/60' : 'text-muted-foreground')}>
                      {new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </p>
                  </div>
                </div>
              );
            })}
            <div ref={bottomRef} />
          </div>

          {/* Input */}
          <form onSubmit={handleSend} className="flex gap-2 border-t bg-muted/30 p-3">
            <input
              className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
              placeholder={`Message ${sellerName}…`}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              maxLength={2000}
              disabled={isPending}
            />
            <Button type="submit" size="icon" disabled={!draft.trim() || isPending}>
              <Send className="h-4 w-4" />
            </Button>
          </form>
        </div>
      )}
    </div>
  );
}
