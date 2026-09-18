import { SandboxDetail } from '../../../src/components/sandbox-detail';

export default function SandboxPage({ params }: { params: { id: string } }) {
  return <SandboxDetail id={params.id} />;
}
