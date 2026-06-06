'use client';

import { useEffect } from 'react';
import { Provider as ReduxProvider } from 'react-redux';
import { QueryClientProvider } from '@tanstack/react-query';
import { store, useAppDispatch } from '@/lib/store';
import { queryClient } from '@/lib/query-client';
import { authApi, applyToken, ApiError } from '@/lib/api';
import { clearCart } from '@/slices/cartSlice';
import { clearCredentials, setUser } from '@/slices/authSlice';
import { Toaster } from '@/components/ui/sonner';

/**
 * Runs once on the client after hydration.
 *
 * Responsibilities:
 * 1. Re-inject the persisted JWT into the API client singleton so that
 *    Authorization headers are sent on every request (cross-origin safe –
 *    no reliance on the httpOnly cookie being forwarded cross-origin).
 * 2. Validate the token against the server and refresh the user profile.
 * 3. Only clear credentials on a definitive 401 – NOT on network errors or
 *    5xx responses, which would cause spurious logouts on flaky connections.
 */
function SessionBootstrap({ children }: { children: React.ReactNode }) {
  const dispatch = useAppDispatch();

  useEffect(() => {
    // Read auth state from the Redux store (already populated from localStorage
    // during store initialisation — no selector subscription needed here).
    const { user, token } = store.getState().auth;

    if (!token) {
      // No JWT in storage. If a user object is present it is a leftover from
      // before the token-persistence refactor; clear it so the user logs in
      // again and gets a fully-persisted session.
      if (user) {
        dispatch(clearCredentials());
        dispatch(clearCart());
      }
      return;
    }

    // Restore the JWT into the API client on every page load.
    applyToken(token);

    let isActive = true;

    authApi
      .getCurrentUser()
      .then((u) => {
        if (!isActive) return;
        dispatch(setUser(u)); // Refresh profile data from server
      })
      .catch((error) => {
        if (!isActive) return;
        // Only log out on a definitive 401 (expired / revoked token).
        // Preserve state on network errors or 5xx so users are not logged
        // out due to transient server problems.
        if (error instanceof ApiError && error.status === 401) {
          applyToken(null);
          dispatch(clearCredentials());
          dispatch(clearCart());
        }
      });

    return () => {
      isActive = false;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- intentional: run once on mount

  return <>{children}</>;
}

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ReduxProvider store={store}>
      <QueryClientProvider client={queryClient}>
        <SessionBootstrap>{children}</SessionBootstrap>
        <Toaster position="top-right" richColors />
      </QueryClientProvider>
    </ReduxProvider>
  );
}
