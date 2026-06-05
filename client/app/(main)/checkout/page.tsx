'use client';

import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useCreateOrder } from '@/hooks/useOrders';
import { useAppSelector, useAppDispatch } from '@/lib/store';
import { clearCart } from '@/slices/cartSlice';
import OrderSummary from '@/components/shared/OrderSummary';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import Link from 'next/link';

export default function CheckoutPage() {
  const router = useRouter();
  const dispatch = useAppDispatch();
  const items = useAppSelector((s) => s.cart.items);
  const { mutate: createOrder, isPending } = useCreateOrder();

  if (items.length === 0) {
    return (
      <div className="container mx-auto px-4 py-20 text-center">
        <p className="text-muted-foreground">Your cart is empty.</p>
        <Button className="mt-4" asChild>
          <Link href="/">Browse books</Link>
        </Button>
      </div>
    );
  }

  function handleConfirm() {
    createOrder(undefined, {
      onSuccess: () => {
        dispatch(clearCart());
        toast.success('Order placed successfully!');
        router.push('/orders');
      },
      onError: (err) => toast.error(err.message ?? 'Failed to place order'),
    });
  }

  return (
    <div className="container mx-auto max-w-lg px-4 py-8">
      <h1 className="mb-6 text-3xl font-bold">Checkout</h1>

      <div className="space-y-6">
        <OrderSummary items={items} />

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Payment</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Payment is arranged directly with the seller after placing your order.
            </p>
          </CardContent>
        </Card>

        <Button className="w-full" size="lg" onClick={handleConfirm} disabled={isPending}>
          {isPending ? 'Placing order…' : 'Confirm order'}
        </Button>

        <Button variant="ghost" className="w-full" asChild>
          <Link href="/cart">Back to cart</Link>
        </Button>
      </div>
    </div>
  );
}
