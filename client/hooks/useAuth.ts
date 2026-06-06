'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { authApi, applyToken, type LoginInput, type RegisterInput } from '@/lib/api';
import { useAppDispatch } from '@/lib/store';
import { setCredentials, setUser, clearCredentials } from '@/slices/authSlice';
import { clearCart } from '@/slices/cartSlice';

export function useLogin() {
  const dispatch = useAppDispatch();
  const router = useRouter();

  return useMutation({
    mutationFn: (data: LoginInput) => authApi.login(data),
    onSuccess: (res) => {
      if (res.user && res.token) {
        dispatch(setCredentials({ user: res.user, token: res.token, expiresAt: res.expires_at }));
        applyToken(res.token);
      } else if (res.user) {
        // Fallback: no token in response (should not happen with current server)
        dispatch(setUser(res.user));
      }
      router.push('/');
    },
  });
}

export function useRegister() {
  const router = useRouter();

  return useMutation({
    mutationFn: (data: RegisterInput) => authApi.register(data),
    onSuccess: () => {
      router.push('/login?registered=1');
    },
  });
}

export function useLogout() {
  const dispatch = useAppDispatch();
  const router = useRouter();

  return useMutation({
    mutationFn: () => authApi.logout(),
    onSettled: () => {
      applyToken(null);
      dispatch(clearCredentials());
      dispatch(clearCart());
      router.push('/login');
    },
  });
}
