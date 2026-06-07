'use client';

import Link from 'next/link';
import { toast } from 'sonner';
import { useMyListings, useDeleteListing } from '@/hooks/useListings';
import { BookListing } from '@/lib/types';
import ConditionBadge from '@/components/shared/ConditionBadge';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Plus, Pencil, Trash2 } from 'lucide-react';

const STATUS_LABELS: Record<BookListing['status'], string> = {
  active: 'Active',
  pending: 'Reviewing',
  reserved: 'Reserved',
  sold: 'Sold',
  delisted: 'Delisted',
};

const STATUS_VARIANTS: Record<
  BookListing['status'],
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  active: 'default',
  pending: 'secondary',
  reserved: 'secondary',
  sold: 'outline',
  delisted: 'destructive',
};

export default function MyListingsPage() {
  const { data: listings, isLoading, isError } = useMyListings();
  const { mutate: deleteListing, isPending: deleting } = useDeleteListing();

  function handleDelete(id: string, title: string) {
    if (!confirm(`Remove "${title}" from your listings?`)) return;
    deleteListing(id, {
      onSuccess: () => toast.success('Listing removed'),
      onError: () => toast.error('Failed to remove listing'),
    });
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-bold">My Listings</h1>
        <Button asChild>
          <Link href="/my-listings/new">
            <Plus className="mr-2 h-4 w-4" />
            New listing
          </Link>
        </Button>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      )}

      {isError && <p className="text-muted-foreground">Failed to load your listings.</p>}

      {listings && listings.length === 0 && (
        <div className="py-20 text-center">
          <p className="text-muted-foreground">You have no listings yet.</p>
          <Button className="mt-4" asChild>
            <Link href="/my-listings/new">Create your first listing</Link>
          </Button>
        </div>
      )}

      {listings && listings.length > 0 && (
        <div className="divide-y rounded-lg border">
          {listings.map((listing) => (
            <div key={listing.id} className="flex items-center gap-4 p-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="truncate font-medium">{listing.title}</p>
                  {listing.status === 'pending' && (
                    <Badge variant="outline" className="text-[10px] uppercase tracking-wider">
                      AI Processing
                    </Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground">{listing.author}</p>
              </div>
              <span className="font-medium">NT${listing.price.toFixed(0)}</span>
              <ConditionBadge condition={listing.condition} />
              <Badge variant={STATUS_VARIANTS[listing.status]}>
                {STATUS_LABELS[listing.status]}
              </Badge>
              <div className="flex gap-1">
                <Button variant="ghost" size="icon" asChild>
                  <Link href={`/my-listings/${listing.id}/edit`}>
                    <Pencil className="h-4 w-4" />
                  </Link>
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleDelete(listing.id, listing.title)}
                  disabled={deleting}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
