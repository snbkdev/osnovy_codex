export async function checkHealth(): Promise<boolean> {
 const response = await fetch("/api/v1/health", { cache: "no-store", signal: AbortSignal.timeout(5000) });
 if (!response.ok) throw new Error(`API returned ${response.status}`);
 const data: unknown = await response.json();
 return typeof data === "object" && data !== null && "status" in data && data.status === "ok";
}
