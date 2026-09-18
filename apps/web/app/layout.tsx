import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'haedes — Give your AI agent a computer on AWS',
  description: 'Disposable AWS computers for agents that need to run real work.',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
