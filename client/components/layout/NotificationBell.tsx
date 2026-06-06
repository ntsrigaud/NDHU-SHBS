'use client';

import Link from 'next/link';
import { Bell } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  useNotifications,
  useNotificationUnreadCount,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
} from '@/hooks/useNotifications';
import { cn } from '@/lib/utils';

const NOTIFICATION_LINKS: Record<string, (payload: Record<string, unknown>) => string> = {
  new_message: (p) => `/messages/${p.listing_id}`,
  order_confirmed: (_p) => `/orders`,
  listing_sold: (p) => `/my-listings/${p.listing_id}/edit`,
};

function notificationText(type: string, payload: Record<string, unknown>): string {
  switch (type) {
    case 'new_message':
      return `New message about "${payload.listing_title ?? 'a listing'}"`;
    case 'order_confirmed':
      return 'Your order has been confirmed';
    case 'listing_sold':
      return `Your listing "${payload.listing_title ?? ''}" was sold`;
    default:
      return 'New notification';
  }
}

export default function NotificationBell() {
  const { data: unread } = useNotificationUnreadCount();
  const { data: notifications } = useNotifications();
  const { mutate: markRead } = useMarkNotificationRead();
  const { mutate: markAll } = useMarkAllNotificationsRead();

  const unreadCount = unread?.count ?? 0;
  const recent = notifications?.slice(0, 8) ?? [];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="relative outline-none focus-visible:ring-2 focus-visible:ring-primary-foreground focus-visible:ring-offset-2">
          <Button variant="ghost" size="icon" className="text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground" asChild>
            <span>
              <Bell className="h-5 w-5" />
            </span>
          </Button>
          {unreadCount > 0 && (
            <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-accent text-[10px] font-semibold text-accent-foreground">
              {unreadCount > 9 ? '9+' : unreadCount}
            </span>
          )}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-80">
        <div className="flex items-center justify-between px-3 py-2">
          <span className="text-sm font-semibold">Notifications</span>
          {unreadCount > 0 && (
            <button
              onClick={() => markAll()}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Mark all read
            </button>
          )}
        </div>
        <DropdownMenuSeparator />

        {recent.length === 0 && (
          <div className="px-3 py-6 text-center text-sm text-muted-foreground">
            No notifications
          </div>
        )}

        {recent.map((n) => {
          const href = NOTIFICATION_LINKS[n.type]?.(n.payload) ?? '/';
          return (
            <DropdownMenuItem key={n.id} asChild>
              <Link
                href={href}
                onClick={() => !n.is_read && markRead(n.id)}
                className={cn(
                  'flex flex-col items-start gap-0.5 px-3 py-2',
                  !n.is_read && 'bg-primary/5'
                )}
              >
                <span className="text-sm">{notificationText(n.type, n.payload)}</span>
                <span className="text-xs text-muted-foreground">
                  {new Date(n.created_at).toLocaleString()}
                </span>
              </Link>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
