import type { Metadata } from 'next';
import { MotionProvider } from '../src/components/motion';
import './globals.css';

export const metadata: Metadata = {
  title: 'Sandbox dashboard — haedes',
  description: 'Human observability for haedes agent computers.',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body><MotionProvider>{children}</MotionProvider></body>
    </html>
  );
}
