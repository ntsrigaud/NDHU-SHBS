'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';

interface Props {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function GlobalError({ error, reset }: Props) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <p className="text-8xl font-bold text-muted-foreground/30">500</p>
      <h1 className="text-2xl font-semibold">Something went wrong</h1>
      <p className="max-w-sm text-muted-foreground">
        An unexpected error occurred. You can try again or go back to the marketplace.
      </p>
      <div className="flex gap-3">
        <Button onClick={reset}>Try again</Button>
        <Button variant="outline" onClick={() => (window.location.href = '/')}>
          Go home
        </Button>
      </div>
      {error.digest && <p className="text-xs text-muted-foreground">Error ID: {error.digest}</p>}
    </div>
  );
}
