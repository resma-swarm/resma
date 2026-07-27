import { useState, useEffect } from "react";

const SERVICES = [
  { name: "dotnet-api", url: "http://localhost:5001" },
  { name: "node-api", url: "http://localhost:5002" },
  { name: "python-api", url: "http://localhost:5003" },
];

interface ServiceStatus {
  name: string;
  status: string;
  time?: string;
  error?: string;
}

export default function App() {
  const [statuses, setStatuses] = useState<ServiceStatus[]>([]);

  async function checkAll() {
    const results = await Promise.all(
      SERVICES.map(async (s) => {
        try {
          const res = await fetch(`${s.url}/health`);
          const data = await res.json();
          return { name: s.name, status: data.status, time: data.ts };
        } catch (e) {
          return { name: s.name, status: "offline", error: String(e) };
        }
      })
    );
    setStatuses(results);
  }

  async function stressTest(service: string, url: string) {
    await fetch(`${url}/cpu?loops=50000`);
    await fetch(`${url}/memory?mb=50`);
  }

  useEffect(() => {
    checkAll();
    const interval = setInterval(checkAll, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", padding: "2rem", background: "#0f172a", color: "#e2e8f0", minHeight: "100vh" }}>
      <h1>Test Dashboard — RESMA Lab</h1>
      <p>Ambiente simulado para coleta de métricas</p>

      <div style={{ display: "grid", gap: "1rem", marginTop: "2rem" }}>
        {statuses.map((s) => {
          const svc = SERVICES.find((x) => x.name === s.name);
          return (
            <div key={s.name} style={{ border: "1px solid #334155", borderRadius: "8px", padding: "1rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <h3 style={{ margin: 0 }}>{s.name}</h3>
                <span style={{
                  padding: "2px 8px",
                  borderRadius: "4px",
                  background: s.status === "healthy" ? "#16a34a" : "#dc2626",
                  fontSize: "0.8rem",
                }}>
                  {s.status}
                </span>
              </div>
              {s.time && <p style={{ fontSize: "0.8rem", color: "#94a3b8" }}>{s.time}</p>}
              <button
                onClick={() => svc && stressTest(svc.name, svc.url)}
                style={{
                  marginTop: "0.5rem",
                  padding: "4px 12px",
                  background: "#3b82f6",
                  border: "none",
                  borderRadius: "4px",
                  color: "white",
                  cursor: "pointer",
                }}
              >
                Stress Test
              </button>
            </div>
          );
        })}
      </div>

      <button
        onClick={checkAll}
        style={{ marginTop: "1rem", padding: "6px 16px", background: "#6366f1", border: "none", borderRadius: "4px", color: "white", cursor: "pointer" }}
      >
        Refresh
      </button>
    </div>
  );
}
