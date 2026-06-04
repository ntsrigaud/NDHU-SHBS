import type { Metadata } from 'next';
import '../styles/globals.css';
import { GeistSans } from 'geist/font/sans';
import Providers from '@/components/layout/Providers';

export const metadata: Metadata = {
  title: 'NDHU Second-Hand Book Store',
  description: 'Campus-exclusive second-hand textbook marketplace for NDHU students.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-TW" className={GeistSans.className}>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
