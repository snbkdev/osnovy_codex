import { MonitorHistoryPage } from "@/components/monitor-history";

export default async function HistoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <MonitorHistoryPage id={id} />;
}
