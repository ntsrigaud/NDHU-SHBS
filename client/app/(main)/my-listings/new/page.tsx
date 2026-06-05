'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useCreateListing } from '@/hooks/useListings';
import { Condition } from '@/lib/types';
import ListingForm from '@/components/shared/ListingForm';
import ImageUpload from '@/components/shared/ImageUpload';

export default function NewListingPage() {
  const router = useRouter();
  const { mutate: createListing, isPending } = useCreateListing();
  const [imageIds, setImageIds] = useState<string[]>([]);
  const [aiCondition, setAiCondition] = useState<{ condition: Condition; confidence: number } | null>(null);

  return (
    <div className="container mx-auto max-w-2xl px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">Create listing</h1>

      <div className="mb-6">
        <p className="mb-2 text-sm font-medium">Photos (up to 6)</p>
        <ImageUpload
          onChange={setImageIds}
          onAiResult={(result) => {
            setAiCondition(result);
            toast.info(
              `AI suggests: ${result.condition} (${Math.round(result.confidence * 100)}%)`,
            );
          }}
        />
      </div>

      <ListingForm
        onSubmit={(values) =>
          createListing(
            { ...values, image_ids: imageIds },
            {
              onSuccess: () => {
                toast.success('Listing created!');
                router.push('/my-listings');
              },
              onError: (err) => toast.error(err.message ?? 'Failed to create listing'),
            },
          )
        }
        isPending={isPending}
        submitLabel="Create listing"
        aiCondition={aiCondition}
      />
    </div>
  );
}
