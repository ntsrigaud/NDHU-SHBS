'use client';

import Image from 'next/image';
import { useOrders } from '@/hooks/useOrders';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import Link from 'next/link';
import { Button } from '@/components/ui/button';

export default function OrdersPage() {
  const { data: orders, isLoading, isError } = useOrders();

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">My Orders</h1>

      {isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-32 w-full rounded-lg" />
          ))}
        </div>
      )}

      {isError && <p className="text-muted-foreground">Failed to load orders.</p>}

      {orders && orders.length === 0 && (
        <div className="py-20 text-center">
          <p className="text-muted-foreground">No orders yet.</p>
          <Button className="mt-4" asChild>
            <Link href="/">Browse books</Link>
          </Button>
        </div>
      )}

      {orders && orders.length > 0 && (
        <div className="space-y-4">
          {orders.map((order) => (
            <div key={order.id} className="rounded-lg border p-4">
              <div className="mb-3 flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">
                    Order #{order.id.slice(0, 8).toUpperCase()}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(order.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="text-right">
                  <Badge variant="secondary" className="capitalize">
                    {order.status}
                  </Badge>
                  <p className="mt-1 font-semibold">NT${order.total.toFixed(0)}</p>
                </div>
              </div>

              <Separator className="mb-3" />

              <div className="space-y-2">
                {order.items.map((item) => (
                  <div key={item.listing_id} className="flex items-center gap-3">
                    <div className="relative h-10 w-10 flex-shrink-0 overflow-hidden rounded bg-muted">
                      {item.image_url ? (
                        <Image
                          src={item.image_url}
                          alt={item.title}
                          fill
                          className="object-cover"
                          sizes="40px"
                        />
                      ) : (
                        <span className="flex h-full items-center justify-center text-lg">📚</span>
                      )}
                    </div>
                    <span className="min-w-0 flex-1 truncate text-sm">{item.title}</span>
                    <span className="text-sm">NT${item.price.toFixed(0)}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
