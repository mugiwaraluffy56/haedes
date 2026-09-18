import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Sandbox dashboard — haedes',
  description: 'Human observability for haedes agent computers.',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
