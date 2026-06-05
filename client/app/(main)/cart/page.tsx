'use client';

import Link from 'next/link';
import { useCart } from '@/hooks/useCart';
import { useAppSelector } from '@/lib/store';
import CartItem from '@/components/shared/CartItem';
import OrderSummary from '@/components/shared/OrderSummary';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';

export default function CartPage() {
  const { isLoading } = useCart();
  const items = useAppSelector((s) => s.cart.items);

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">Cart</h1>

      {isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      )}

      {!isLoading && items.length === 0 && (
        <div className="py-20 text-center">
          <p className="text-muted-foreground">Your cart is empty.</p>
          <Button className="mt-4" asChild>
            <Link href="/">Browse books</Link>
          </Button>
        </div>
      )}

      {!isLoading && items.length > 0 && (
        <div className="grid gap-8 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <div className="divide-y">
              {items.map((item) => (
                <CartItem key={item.listingId} item={item} />
              ))}
            </div>
            <Separator className="my-4" />
          </div>

          <div className="space-y-4">
            <OrderSummary items={items} />
            <Button className="w-full" asChild>
              <Link href="/checkout">Proceed to checkout</Link>
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
