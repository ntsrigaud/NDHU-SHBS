import Link from 'next/link';
import Image from 'next/image';
import { BookListing } from '@/lib/types';
import { Card, CardContent } from '@/components/ui/card';
import ConditionBadge from './ConditionBadge';

export default function BookCard({ listing }: { listing: BookListing }) {
  const cover = listing.images?.[0]?.cdn_url;

  return (
    <Link href={`/books/${listing.id}`}>
      <Card className="group overflow-hidden transition-shadow hover:shadow-md">
        <div className="relative aspect-[3/4] w-full bg-muted">
          {cover ? (
            <Image
              src={cover}
              alt={listing.title}
              fill
              className="object-cover transition-transform group-hover:scale-105"
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
            />
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              <span className="text-4xl">📚</span>
            </div>
          )}
        </div>
        <CardContent className="p-3">
          <p className="line-clamp-2 text-sm font-medium leading-tight">{listing.title}</p>
          <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">{listing.author}</p>
          <div className="mt-2 flex items-center justify-between">
            <span className="font-semibold">NT${listing.price.toFixed(0)}</span>
            <ConditionBadge condition={listing.condition} />
          </div>
          {listing.department && (
            <p className="mt-1 truncate text-xs text-muted-foreground">{listing.department}</p>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}
