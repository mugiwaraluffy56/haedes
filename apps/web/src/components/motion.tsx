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

export function FadeIn({ children, className, delay = 0, ariaLabel }: { children: ReactNode; className?: string; delay?: number; ariaLabel?: string }) {
  return (
    <m.div
      className={className}
      aria-label={ariaLabel}
      initial={{ opacity: 0, y: 18 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.48, delay, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </m.div>
  );
}

export function MotionToggle() {
  return <button className="motion-toggle" type="button" onClick={() => document.documentElement.toggleAttribute('data-motion-paused')} aria-label="Pause or resume ambient motion">Motion</button>;
}
