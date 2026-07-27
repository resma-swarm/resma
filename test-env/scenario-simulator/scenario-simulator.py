import os
import time
import random

import requests

LEAKY_URL = os.getenv("LEAKY_URL", "http://leaky-service:8080")
OOM_URL = os.getenv("OOM_URL", "http://oom-prone-service:8080")
DRIFT_URL = os.getenv("DRIFT_URL", "http://drift-service:8080")

CYCLE_INTERVAL = float(os.getenv("CYCLE_INTERVAL", "8"))

# OOM scheduler independente do ciclo de leak/drift.
# Intervalo base sorteado com jitter amplo -> OOMs esparcidos e irregulares.
# Default ~10 min de media (6 OOMs/hora), com intervalos entre 300s e 900s.
OOM_BASE_INTERVAL = float(os.getenv("OOM_INTERVAL", "600"))
OOM_JITTER = float(os.getenv("OOM_JITTER", "0.5"))  # 0.5 = [0.5x, 1.5x] do base
# Cooldown extra apos um OOM real para o container reiniciar e passar o healthcheck.
OOM_COOLDOWN = float(os.getenv("OOM_COOLDOWN", "60"))


def trigger_leak():
    try:
        mb = random.randint(2, 8)
        requests.get(f"{LEAKY_URL}/leak?mb={mb}", timeout=10)
        print(f"[leak] requested {mb}MB on leaky-service")
    except Exception as e:
        print(f"[leak] error: {e}")


def trigger_oom():
    # Piso acima do limite do container (64M) para garantir que cada disparo
    # seja um OOM real, nao apenas acumulacao de estado.
    mb = random.randint(80, 160)
    try:
        requests.get(f"{OOM_URL}/oom?mb={mb}", timeout=15)
        print(f"[oom] requested {mb}MB burst on oom-prone-service (no OOM?)")
        return False
    except Exception as e:
        print(f"[oom] service killed (expected): {e}")
        return True


def trigger_drift():
    try:
        mode = random.choice(["idle", "busy", "mixed"])
        requests.get(f"{DRIFT_URL}/mode?mode={mode}", timeout=10)
        print(f"[drift] set mode={mode} on drift-service")
    except Exception as e:
        print(f"[drift] error: {e}")


def schedule_next_oom(extra_cooldown=0.0):
    factor = random.uniform(1 - OOM_JITTER, 1 + OOM_JITTER)
    wait = max(30.0, OOM_BASE_INTERVAL * factor + extra_cooldown)
    return wait


def main():
    print("=== RESMA Scenario Simulator ===")
    print(f"  leaky-service: {LEAKY_URL}")
    print(f"  oom-prone-service: {OOM_URL}")
    print(f"  drift-service: {DRIFT_URL}")
    print(f"  cycle interval: {CYCLE_INTERVAL}s")
    print(f"  oom interval: {OOM_BASE_INTERVAL}s (jitter +/-{OOM_JITTER*100:.0f}%, cooldown {OOM_COOLDOWN}s)")
    print()

    cycle = 0
    next_oom_in = schedule_next_oom()
    print(f"[oom] first OOM scheduled in {next_oom_in:.0f}s")

    while True:
        cycle += 1
        print(f"\n--- cycle {cycle} ---")

        trigger_leak()

        if cycle % 5 == 0:
            trigger_drift()

        # Scheduler de OOM independente: conta o tempo do ciclo contra o
        # proximo disparo agendado, em vez de cycle % 3.
        next_oom_in -= CYCLE_INTERVAL
        if next_oom_in <= 0:
            killed = trigger_oom()
            cooldown = OOM_COOLDOWN if killed else 0.0
            next_oom_in = schedule_next_oom(extra_cooldown=cooldown)
            print(f"[oom] next OOM scheduled in {next_oom_in:.0f}s")

        jitter = random.uniform(-2, 2)
        time.sleep(max(2, CYCLE_INTERVAL + jitter))


if __name__ == "__main__":
    main()
