'use client';

import { useParams } from 'next/navigation';
import { useBook } from '@/hooks/useBooks';
import { useAppSelector } from '@/lib/store';
import MessageThread from '@/components/shared/MessageThread';
import { Skeleton } from '@/components/ui/skeleton';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export default function ConversationPage() {
  const { listingId } = useParams<{ listingId: string }>();
  const { data: listing, isLoading } = useBook(listingId);
  const currentUser = useAppSelector((s) => s.auth.user);

  const otherUserName =
    listing?.seller && listing.seller.id !== currentUser?.id ? listing.seller.name : 'Buyer';

  return (
    <div
      className="container mx-auto flex max-w-2xl flex-col px-4 py-8"
      style={{ height: 'calc(100vh - 56px)' }}
    >
      <Link
        href="/messages"
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        All messages
      </Link>

      {listing && (
        <p className="mb-3 text-sm text-muted-foreground">
          Re:{' '}
          <Link href={`/books/${listingId}`} className="underline underline-offset-4">
            {listing.title}
          </Link>
        </p>
      )}

      {isLoading ? (
        <div className="flex-1 space-y-3 p-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-2/3" />
          ))}
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-hidden rounded-lg border">
          <MessageThread listingId={listingId} otherUserName={otherUserName} />
        </div>
      )}
    </div>
  );
}
