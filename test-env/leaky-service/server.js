const express = require('express')
const app = express()
const port = 8080

let leakBuffer = []

app.get('/health', (req, res) => res.json({ status: 'ok' }))

app.get('/leak', (req, res) => {
  const mb = parseInt(req.query.mb) || 5
  const chunk = Buffer.alloc(mb * 1024 * 1024, 'x')
  leakBuffer.push(chunk)
  res.json({ leaked: mb, total_mb: Math.round(process.memoryUsage().rss / 1024 / 1024) })
})

app.get('/reset', (req, res) => {
  leakBuffer = []
  res.json({ cleared: true })
})

app.listen(port, () => console.log(`leaky-service on :${port}`))
