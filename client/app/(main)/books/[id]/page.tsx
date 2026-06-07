'use client';

import { useParams, useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useBook } from '@/hooks/useBooks';
import { useAppDispatch, useAppSelector } from '@/lib/store';
import { addItem } from '@/slices/cartSlice';
import ImageCarousel from '@/components/shared/ImageCarousel';
import ConditionBadge from '@/components/shared/ConditionBadge';
import InlineChatPanel from '@/components/shared/InlineChatPanel';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { ShoppingCart, ArrowLeft, MessageSquare } from 'lucide-react';
import Link from 'next/link';

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const dispatch = useAppDispatch();
  const user = useAppSelector((s) => s.auth.user);
  const cartItems = useAppSelector((s) => s.cart.items);

  const { data: listing, isLoading, isError } = useBook(id);

  if (isLoading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="grid gap-8 md:grid-cols-2">
          <Skeleton className="aspect-square w-full rounded-lg" />
          <div className="space-y-4">
            <Skeleton className="h-8 w-3/4" />
            <Skeleton className="h-5 w-1/2" />
            <Skeleton className="h-10 w-32" />
            <Skeleton className="h-24 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (isError || !listing) {
    return (
      <div className="container mx-auto px-4 py-20 text-center">
        <p className="text-muted-foreground">Listing not found.</p>
        <Button variant="ghost" className="mt-4" onClick={() => router.push('/')}>
          Back to marketplace
        </Button>
      </div>
    );
  }

  const alreadyInCart = cartItems.some((i) => i.listingId === listing.id);
  const isSold = listing.status === 'sold' || listing.status === 'delisted';
  const isOwnListing = user?.id === listing.seller_id;
  const sellerName = listing.seller?.name ?? 'Seller';

  function handleAddToCart() {
    if (!user) {
      router.push(`/login?next=/books/${id}`);
      return;
    }
    if (alreadyInCart) return;
    dispatch(
      addItem({
        listingId: listing!.id,
        title: listing!.title,
        price: listing!.price,
        imageUrl: listing!.images?.[0]?.cdn_url ?? '',
        sellerName,
      }),
    );
    toast.success('Added to cart');
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <Link
        href="/"
        className="mb-6 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to marketplace
      </Link>

      <div className="grid gap-8 md:grid-cols-2">
        <ImageCarousel images={listing.images ?? []} />

        <div className="space-y-4">
          <div>
            <h1 className="text-2xl font-bold leading-tight">{listing.title}</h1>
            <p className="mt-1 text-muted-foreground">{listing.author}</p>
          </div>

          <div className="flex items-center gap-3">
            <span className="text-3xl font-bold text-accent">NT${listing.price.toFixed(0)}</span>
            <ConditionBadge condition={listing.condition} />
            {listing.status === 'reserved' && <Badge variant="secondary">Reserved</Badge>}
          </div>

          <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
            {listing.department && <span>{listing.department}</span>}
            {listing.course_code && <span>· {listing.course_code}</span>}
            {listing.isbn && <span>· ISBN: {listing.isbn}</span>}
          </div>

          {listing.ai_confidence != null && (
            <p className="text-xs text-muted-foreground">
              AI condition confidence: {Math.round(listing.ai_confidence * 100)}%
            </p>
          )}

          <Separator />

          {listing.description && (
            <div>
              <h2 className="mb-1 font-semibold">Description</h2>
              <p className="whitespace-pre-wrap text-sm text-muted-foreground">
                {listing.description}
              </p>
            </div>
          )}

          {/* Action buttons */}
          <div className="flex gap-3 pt-2">
            <Button
              className="flex-1"
              onClick={handleAddToCart}
              disabled={isSold || alreadyInCart}
            >
              <ShoppingCart className="mr-2 h-4 w-4" />
              {isSold ? 'Sold' : alreadyInCart ? 'In cart' : 'Add to cart'}
            </Button>

            {/* Redirect unauthenticated users to login */}
            {!user && !isSold && (
              <Button variant="outline" onClick={() => router.push(`/login?next=/books/${id}`)}>
                <MessageSquare className="mr-2 h-4 w-4" />
                Message seller
              </Button>
            )}
          </div>

          {listing.seller && (
            <p className="text-sm text-muted-foreground">
              Listed by <span className="font-medium">{sellerName}</span>
            </p>
          )}

          {/* Inline chat — shown only to logged-in buyers */}
          {user && !isOwnListing && !isSold && (
            <InlineChatPanel listingId={listing.id} sellerName={sellerName} />
          )}

          {/* Own listing notice */}
          {isOwnListing && (
            <div className="rounded-lg border border-dashed p-3 text-center text-sm text-muted-foreground">
              This is your listing.{' '}
              <Link href="/messages" className="font-medium text-primary underline underline-offset-4">
                View messages from buyers →
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
