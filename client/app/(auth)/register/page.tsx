'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'sonner';
import { useRegister } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

const passwordRules = z
  .string()
  .min(8, 'At least 8 characters')
  .regex(/[A-Z]/, 'At least one uppercase letter')
  .regex(/[a-z]/, 'At least one lowercase letter')
  .regex(/[0-9]/, 'At least one number')
  .regex(/[^A-Za-z0-9]/, 'At least one special character');

const registerSchema = z
  .object({
    name: z.string().min(1, 'Name is required').max(100),
    email: z.string().email('Please enter a valid email'),
    password: passwordRules,
    confirmPassword: z.string().min(1, 'Please confirm your password'),
  })
  .refine((v) => v.password === v.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  });

type RegisterValues = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const router = useRouter();
  const { mutate: register, isPending, error } = useRegister();

  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { name: '', email: '', password: '', confirmPassword: '' },
  });

  function onSubmit({ name, email, password }: RegisterValues) {
    register(
      { name, email, password },
      {
        onSuccess: () => {
          toast.success('Account created! Check your email to verify your address.');
          router.push('/login?registered=1');
        },
        onError: (err) => {
          toast.error(err.message ?? 'Registration failed. Please try again.');
        },
      }
    );
  }

  return (
    <Card className="w-full max-w-sm">
      <CardHeader className="text-center">
        <CardTitle className="text-2xl">Create an account</CardTitle>
        <CardDescription>Join the NDHU second-hand book marketplace</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Full name</FormLabel>
                  <FormControl>
                    <Input placeholder="Jane Smith" autoComplete="name" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      type="email"
                      placeholder="you@example.com"
                      autoComplete="email"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder="••••••••"
                      autoComplete="new-password"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="confirmPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Confirm password</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder="••••••••"
                      autoComplete="new-password"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {error && (
              <p className="text-sm text-destructive">{error.message ?? 'Registration failed'}</p>
            )}

            <Button type="submit" className="w-full" disabled={isPending}>
              {isPending ? 'Creating account…' : 'Create account'}
            </Button>
          </form>
        </Form>

        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{' '}
          <Link href="/login" className="font-medium underline underline-offset-4">
            Sign in
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
