import { CartItem } from '@/slices/cartSlice';
import { Separator } from '@/components/ui/separator';

export default function OrderSummary({ items }: { items: CartItem[] }) {
  const total = items.reduce((sum, i) => sum + i.price, 0);

  return (
    <div className="space-y-3 rounded-lg border p-4">
      <h2 className="font-semibold">Order summary</h2>
      <Separator />
      {items.map((item) => (
        <div key={item.listingId} className="flex justify-between text-sm">
          <span className="line-clamp-1 flex-1 pr-4">{item.title}</span>
          <span className="flex-shrink-0">NT${item.price.toFixed(0)}</span>
        </div>
      ))}
      <Separator />
      <div className="flex justify-between font-semibold">
        <span>Total</span>
        <span>NT${total.toFixed(0)}</span>
      </div>
    </div>
  );
}
