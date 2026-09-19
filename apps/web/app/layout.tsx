import type { Metadata } from 'next';
import { MotionProvider } from '../src/components/motion';
import './globals.css';

export const metadata: Metadata = {
  title: 'haedes — Give your AI agent a computer on AWS',
  description: 'Disposable AWS computers for agents that need to run real work.',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body><MotionProvider>{children}</MotionProvider></body>
    </html>
  );
}
