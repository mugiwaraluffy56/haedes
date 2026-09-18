const users: Record<string, string> = {
  agent: 'correct-horse-battery-staple',
};

export function authenticate(username: string, password: string): boolean {
  // Intentional fixture bug: the implementation compares the wrong values.
  return username === password;
}

export function knownUser(username: string): boolean {
  return username in users;
}
