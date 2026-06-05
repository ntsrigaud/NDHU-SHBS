'use client';

import { useParams, useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useBook } from '@/hooks/useBooks';
import { useUpdateListing } from '@/hooks/useListings';
import ListingForm from '@/components/shared/ListingForm';
import { Skeleton } from '@/components/ui/skeleton';

export default function EditListingPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { data: listing, isLoading } = useBook(id);
  const { mutate: updateListing, isPending } = useUpdateListing(id);

  if (isLoading) {
    return (
      <div className="container mx-auto max-w-2xl space-y-4 px-4 py-8">
        <Skeleton className="h-9 w-48" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (!listing) {
    return (
      <div className="container mx-auto px-4 py-8 text-muted-foreground">
        Listing not found.
      </div>
    );
  }

  return (
    <div className="container mx-auto max-w-2xl px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">Edit listing</h1>
      <ListingForm
        defaultValues={listing}
        onSubmit={(values) =>
          updateListing(values, {
            onSuccess: () => {
              toast.success('Listing updated');
              router.push('/my-listings');
            },
            onError: (err) => toast.error(err.message ?? 'Failed to update listing'),
          })
        }
        isPending={isPending}
        submitLabel="Save changes"
      />
    </div>
  );
}
