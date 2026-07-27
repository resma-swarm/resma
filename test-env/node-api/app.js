const express = require("express");
const { WorkloadSimulator } = require("./workload_sim");
const app = express();
const port = 3000;

const _workload = new WorkloadSimulator({
  maxCpuMs: 50,
  maxMemMb: 8,
  personality: "spike",
});
_workload.start();

app.get("/", (req, res) => {
  res.json({ service: "node-api", status: "running", time: new Date().toISOString() });
});

app.get("/cpu", (req, res) => {
  const loops = parseInt(req.query.loops) || 1000;
  let result = 0;
  for (let i = 0; i < loops; i++) {
    result += Math.sqrt(i) * Math.sin(i);
  }
  res.json({ service: "node-api", result, loops });
});

app.get("/memory", (req, res) => {
  const mb = parseInt(req.query.mb) || 10;
  const buffers = [];
  for (let i = 0; i < mb; i++) {
    buffers.push(Buffer.alloc(1024 * 1024));
  }
  res.json({ service: "node-api", allocatedMB: mb });
});

app.get("/health", (req, res) => {
  res.json({ status: "healthy", ts: new Date().toISOString() });
});

app.listen(port, () => {
  console.log(`node-api running on port ${port}`);
});
