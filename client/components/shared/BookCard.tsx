import Link from 'next/link';
import Image from 'next/image';
import { BookListing } from '@/lib/types';
import ConditionBadge from './ConditionBadge';

export default function BookCard({ listing }: { listing: BookListing }) {
  const cover = listing.images?.[0]?.cdn_url;

  return (
    <Link href={`/books/${listing.id}`} className="group block">
      <div className="overflow-hidden rounded-xl border bg-card shadow-sm transition-all duration-200 hover:-translate-y-1 hover:shadow-md">
        {/* Cover image */}
        <div className="relative aspect-[3/4] w-full overflow-hidden bg-muted">
          {cover ? (
            <Image
              src={cover}
              alt={listing.title}
              fill
              className="object-cover transition-transform duration-300 group-hover:scale-105"
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
            />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
              <span className="text-5xl">📚</span>
              <span className="text-xs">No photo</span>
            </div>
          )}
          {/* Condition badge overlay */}
          <div className="absolute right-2 top-2">
            <ConditionBadge condition={listing.condition} />
          </div>
        </div>

        {/* Details */}
        <div className="p-3">
          <p className="line-clamp-2 text-[13px] font-semibold leading-snug text-foreground">
            {listing.title}
          </p>
          <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">{listing.author}</p>

          <div className="mt-2.5 flex items-center justify-between gap-2">
            <span className="text-base font-bold text-accent">NT${listing.price.toFixed(0)}</span>
            {listing.department && (
              <span className="truncate rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-secondary-foreground">
                {listing.department}
              </span>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
