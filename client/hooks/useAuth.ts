'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { authApi, type LoginInput, type RegisterInput } from '@/lib/api';
import { useAppDispatch } from '@/lib/store';
import { setUser, clearCredentials } from '@/slices/authSlice';
import { clearCart } from '@/slices/cartSlice';

export function useLogin() {
  const dispatch = useAppDispatch();
  const router = useRouter();

  return useMutation({
    mutationFn: (data: LoginInput) => authApi.login(data),
    onSuccess: (res) => {
      if (res.user) {
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
      dispatch(clearCredentials());
      dispatch(clearCart());
      router.push('/login');
    },
  });
}
