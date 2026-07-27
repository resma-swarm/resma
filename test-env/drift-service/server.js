const express = require('express')
const app = express()
const port = 8080

let currentMode = 'idle'
let workBuffer = []

app.get('/health', (req, res) => res.json({ status: 'ok', mode: currentMode }))

app.get('/mode', (req, res) => {
  currentMode = req.query.mode || 'idle'
  workBuffer = []
  res.json({ mode: currentMode })
})

app.get('/cpu', (req, res) => {
  const loops = parseInt(req.query.loops) || 5000
  let x = 0
  const factor = currentMode === 'busy' ? 5 : currentMode === 'mixed' ? 2 : 1
  for (let i = 0; i < loops * factor; i++) {
    x += Math.sqrt(i)
  }
  res.json({ result: x, mode: currentMode })
})

app.get('/memory', (req, res) => {
  const mb = parseInt(req.query.mb) || 5
  const factor = currentMode === 'busy' ? 3 : currentMode === 'mixed' ? 1.5 : 0.5
  const chunk = Buffer.alloc(Math.round(mb * factor * 1024 * 1024), 'x')
  workBuffer.push(chunk)
  if (workBuffer.length > 10) workBuffer.shift()
  res.json({ allocated: Math.round(mb * factor), mode: currentMode })
})

app.listen(port, () => console.log(`drift-service on :${port}`))
