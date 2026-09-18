export const queryKeys = {
  sandboxes: ['sandboxes'] as const,
  sandbox: (id: string) => ['sandboxes', id] as const,
};
