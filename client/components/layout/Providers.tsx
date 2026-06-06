'use client';

import { useEffect } from 'react';
import { Provider as ReduxProvider } from 'react-redux';
import { QueryClientProvider } from '@tanstack/react-query';
import { store, useAppDispatch } from '@/lib/store';
import { queryClient } from '@/lib/query-client';
import { authApi, ApiError } from '@/lib/api';
import { clearCart } from '@/slices/cartSlice';
import { clearCredentials, setUser } from '@/slices/authSlice';
import { Toaster } from '@/components/ui/sonner';

function SessionBootstrap({ children }: { children: React.ReactNode }) {
  const dispatch = useAppDispatch();

  useEffect(() => {
    let isActive = true;

    authApi
      .getCurrentUser()
      .then((user) => {
        if (!isActive) return;
        dispatch(setUser(user));
      })
      .catch((error) => {
        if (!isActive) return;
        if (error instanceof ApiError && error.status === 401) {
          dispatch(clearCredentials());
          dispatch(clearCart());
        }
      });

    return () => {
      isActive = false;
    };
  }, [dispatch]);

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
