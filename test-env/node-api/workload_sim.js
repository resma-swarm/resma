class WorkloadSimulator {
  constructor({
    maxCpuMs = 50,
    maxMemMb = 8,
    cycleSec = 2.5,
    personality = "spike",
  } = {}) {
    this.maxCpuMs = maxCpuMs;
    this.maxMemMb = maxMemMb;
    this.cycleSec = cycleSec;
    this.personality = personality;
    this.memPool = [];
    this.running = true;
    this.startTime = Date.now();
  }

  _intensity(elapsed) {
    const t = elapsed / 1000;
    let base = 0.5 + 0.3 * Math.sin((2 * Math.PI * t) / 300);

    if (this.personality === "spike") {
      const roll = Math.random();
      const spike = roll < 0.08 ? 0.85 : 0.25;
      base = base * 0.35 + spike * 0.65;
    } else if (this.personality === "batch") {
      const burstPhase = t % 60;
      const burst = burstPhase < 15 ? 0.9 : 0.15;
      base = base * 0.25 + burst * 0.75;
    } else if (this.personality === "business") {
      const simHour = (t % 600) / 25;
      let dayFactor;
      if (simHour >= 8 && simHour <= 18) {
        dayFactor = 0.4 + 0.5 * Math.sin((Math.PI * (simHour - 8)) / 10);
      } else {
        dayFactor = 0.12;
      }
      base = base * 0.15 + dayFactor * 0.85;
    }

    base *= 0.8 + Math.random() * 0.4;
    return Math.max(0.05, Math.min(1.0, base));
  }

  _cpuWork(ms) {
    const end = Date.now() + ms;
    while (Date.now() < end) {
      Math.sqrt(Math.random()) * Math.sin(Math.random());
    }
  }

  _adjustMemory(targetMb) {
    const currentMb = this.memPool.length;
    if (targetMb > currentMb) {
      const alloc = Math.min(targetMb - currentMb, 2);
      for (let i = 0; i < alloc; i++) {
        this.memPool.push(Buffer.alloc(1024 * 1024));
      }
    } else if (targetMb < currentMb) {
      const release = Math.max(1, Math.floor((currentMb - targetMb) / 3));
      this.memPool.splice(0, release);
    }
  }

  _touchMemory() {
    if (this.memPool.length > 0) {
      const idx = Math.floor(Math.random() * this.memPool.length);
      this.memPool[idx][Math.floor(Math.random() * 1024)] =
        Math.floor(Math.random() * 256);
    }
  }

  async _run() {
    while (this.running) {
      const elapsed = Date.now() - this.startTime;
      const intensity = this._intensity(elapsed);

      const cpuMs = Math.floor(intensity * this.maxCpuMs);
      this._cpuWork(cpuMs);

      const targetMb = Math.max(1, Math.floor(intensity * this.maxMemMb));
      this._adjustMemory(targetMb);
      this._touchMemory();

      const sleepSec = Math.max(
        0.5,
        this.cycleSec + (Math.random() - 0.5)
      );
      await new Promise((r) => setTimeout(r, sleepSec * 1000));
    }
  }

  start() {
    this._run().catch(console.error);
  }

  stop() {
    this.running = false;
  }
}

module.exports = { WorkloadSimulator };
