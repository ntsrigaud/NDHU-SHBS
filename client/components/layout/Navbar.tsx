'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAppSelector, useAppDispatch } from '@/lib/store';
import { clearCredentials } from '@/slices/authSlice';
import { clearCart } from '@/slices/cartSlice';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { ShoppingCart, MessageSquare } from 'lucide-react';
import { authApi, applyToken } from '@/lib/api';
import { useUnreadCount } from '@/hooks/useMessages';
import NotificationBell from './NotificationBell';

export default function Navbar() {
  const dispatch = useAppDispatch();
  const router = useRouter();
  const user = useAppSelector((s) => s.auth.user);
  const cartCount = useAppSelector((s) => s.cart.items.length);
  const { data: unread } = useUnreadCount();

  async function handleLogout() {
    try {
      await authApi.logout();
    } catch {
      // proceed with local logout even if the server call fails
    }
    applyToken(null);
    dispatch(clearCredentials());
    dispatch(clearCart());
    router.push('/login');
  }

  return (
    <header className="sticky top-0 z-50 w-full shadow-md" style={{ background: 'linear-gradient(135deg, hsl(152 52% 20%) 0%, hsl(158 48% 26%) 100%)' }}>
      <div className="container mx-auto flex h-16 items-center justify-between px-4">
        {/* Brand */}
        <Link href="/" className="group flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-white/15 text-base font-bold text-white ring-1 ring-white/20 transition-all group-hover:bg-white/25">
            東
          </div>
          <div className="leading-tight">
            <p className="text-[15px] font-bold tracking-wide text-white">NDHU 二手書市集</p>
            <p className="text-[10px] font-medium tracking-wider text-white/60 uppercase">Second-Hand Book Store</p>
          </div>
        </Link>

        <nav className="flex items-center gap-1">
          {user ? (
            <>
              <NotificationBell />

              <Link href="/messages" className="relative">
                <Button variant="ghost" size="icon" className="text-white/80 hover:bg-white/10 hover:text-white">
                  <MessageSquare className="h-5 w-5" />
                  {(unread?.count ?? 0) > 0 && (
                    <span className="absolute -right-0.5 -top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-accent text-[10px] font-bold text-white shadow">
                      {unread!.count}
                    </span>
                  )}
                </Button>
              </Link>

              <Link href="/cart" className="relative">
                <Button variant="ghost" size="icon" className="text-white/80 hover:bg-white/10 hover:text-white">
                  <ShoppingCart className="h-5 w-5" />
                  {cartCount > 0 && (
                    <span className="absolute -right-0.5 -top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-accent text-[10px] font-bold text-white shadow">
                      {cartCount}
                    </span>
                  )}
                </Button>
              </Link>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button className="ml-1 rounded-full outline-none transition-transform hover:scale-105 focus-visible:ring-2 focus-visible:ring-white/50">
                    <Avatar className="h-8 w-8 ring-2 ring-white/30">
                      <AvatarFallback className="bg-accent text-white text-sm font-bold">
                        {user.name[0]?.toUpperCase() ?? 'U'}
                      </AvatarFallback>
                    </Avatar>
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-52 shadow-xl">
                  <div className="px-3 py-2.5 border-b">
                    <p className="text-sm font-semibold">{user.name}</p>
                    <p className="truncate text-xs text-muted-foreground">{user.email}</p>
                  </div>
                  <DropdownMenuItem asChild>
                    <Link href="/my-listings">My Listings</Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/orders">My Orders</Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/messages">Messages</Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={handleLogout} className="text-destructive focus:text-destructive">
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            <>
              <Button variant="ghost" className="text-white/80 hover:bg-white/10 hover:text-white" asChild>
                <Link href="/login">Login</Link>
              </Button>
              <Button className="bg-accent hover:bg-accent/90 text-white font-semibold shadow-md" asChild>
                <Link href="/register">Register</Link>
              </Button>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
