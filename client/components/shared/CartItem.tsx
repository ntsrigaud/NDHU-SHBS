'use client';

import Image from 'next/image';
import Link from 'next/link';
import { CartItem as CartItemType } from '@/slices/cartSlice';
import { useRemoveFromCart } from '@/hooks/useCart';
import { Button } from '@/components/ui/button';
import { X } from 'lucide-react';
import { toast } from 'sonner';

export default function CartItem({ item }: { item: CartItemType }) {
  const { mutate: remove, isPending } = useRemoveFromCart();

  function handleRemove() {
    remove(item.listingId, {
      onError: () => toast.error('Failed to remove item'),
    });
  }

  return (
    <div className="flex items-center gap-4 py-4">
      <div className="relative h-16 w-16 flex-shrink-0 overflow-hidden rounded-md bg-muted">
        {item.imageUrl ? (
          <Image src={item.imageUrl} alt={item.title} fill className="object-cover" sizes="64px" />
        ) : (
          <span className="flex h-full items-center justify-center text-2xl">📚</span>
        )}
      </div>

      <div className="min-w-0 flex-1">
        <Link href={`/books/${item.listingId}`} className="line-clamp-2 font-medium hover:underline">
          {item.title}
        </Link>
        <p className="text-sm text-muted-foreground">{item.sellerName}</p>
      </div>

      <span className="font-semibold">NT${item.price.toFixed(0)}</span>

      <Button
        variant="ghost"
        size="icon"
        className="text-muted-foreground"
        onClick={handleRemove}
        disabled={isPending}
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
}
