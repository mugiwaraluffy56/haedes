'use client';

import type { ReactNode } from 'react';
import { LazyMotion, MotionConfig, domAnimation, m } from 'motion/react';

export function MotionProvider({ children }: { children: ReactNode }) {
  return (
    <LazyMotion features={domAnimation}>
      <MotionConfig reducedMotion="user">{children}</MotionConfig>
    </LazyMotion>
  );
}

export function FadeIn({ children, className, delay = 0 }: { children: ReactNode; className?: string; delay?: number }) {
  return (
    <m.div
      className={className}
      initial={{ opacity: 0, y: 14 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.38, delay, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </m.div>
  );
}

export function MotionToggle() {
  return <button className="motion-toggle" type="button" onClick={() => document.documentElement.toggleAttribute('data-motion-paused')} aria-label="Pause or resume ambient motion">Motion</button>;
}
