import math
import random
import threading
import time


class WorkloadSimulator:

    def __init__(
        self,
        max_cpu_ms=60,
        max_mem_mb=10,
        cycle_sec=2.5,
        personality="batch",
    ):
        self.max_cpu_ms = max_cpu_ms
        self.max_mem_mb = max_mem_mb
        self.cycle_sec = cycle_sec
        self.personality = personality
        self._mem_pool = []
        self._running = True
        self._start_time = time.time()

    def _intensity(self, elapsed):
        base = 0.5 + 0.3 * math.sin(2 * math.pi * elapsed / 300)

        if self.personality == "batch":
            burst_phase = elapsed % 60
            burst = 0.9 if burst_phase < 15 else 0.15
            base = base * 0.25 + burst * 0.75

        elif self.personality == "spike":
            roll = random.random()
            spike = 0.85 if roll < 0.08 else 0.25
            base = base * 0.35 + spike * 0.65

        elif self.personality == "business":
            sim_hour = (elapsed % 600) / 25
            if 8 <= sim_hour <= 18:
                day_factor = 0.4 + 0.5 * math.sin(
                    math.pi * (sim_hour - 8) / 10
                )
            else:
                day_factor = 0.12
            base = base * 0.15 + day_factor * 0.85

        base *= random.uniform(0.8, 1.2)
        return max(0.05, min(1.0, base))

    def _cpu_work(self, ms):
        end = time.time() + ms / 1000.0
        while time.time() < end:
            _ = math.sqrt(random.random()) * math.sin(random.random())

    def _adjust_memory(self, target_mb):
        current_mb = len(self._mem_pool)
        if target_mb > current_mb:
            alloc = min(target_mb - current_mb, 2)
            for _ in range(alloc):
                self._mem_pool.append(bytearray(1024 * 1024))
        elif target_mb < current_mb:
            release = max(1, (current_mb - target_mb) // 3)
            self._mem_pool = self._mem_pool[release:]

    def _touch_memory(self):
        if self._mem_pool:
            idx = random.randint(0, len(self._mem_pool) - 1)
            self._mem_pool[idx][random.randint(0, 1023)] = random.randint(0, 255)

    def _run(self):
        while self._running:
            elapsed = time.time() - self._start_time
            intensity = self._intensity(elapsed)

            cpu_ms = int(intensity * self.max_cpu_ms)
            self._cpu_work(cpu_ms)

            target_mb = max(1, int(intensity * self.max_mem_mb))
            self._adjust_memory(target_mb)
            self._touch_memory()

            sleep_sec = self.cycle_sec + random.uniform(-0.5, 0.5)
            time.sleep(max(0.5, sleep_sec))

    def start(self):
        t = threading.Thread(target=self._run, daemon=True)
        t.start()

    def stop(self):
        self._running = False
