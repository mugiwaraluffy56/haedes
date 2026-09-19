import { SandboxDetail } from '../../../src/components/sandbox-detail';

export default async function SandboxPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <SandboxDetail id={id} />;
}
