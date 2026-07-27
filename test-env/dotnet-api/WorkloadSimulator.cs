using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;

namespace dotnet_api;

public class WorkloadSimulator
{
    private readonly int _maxCpuMs;
    private readonly int _maxMemMb;
    private readonly double _cycleSec;
    private readonly string _personality;
    private readonly List<byte[]> _memPool = new();
    private readonly DateTime _startTime = DateTime.UtcNow;
    private readonly Random _rng = new();
    private volatile bool _running = true;

    public WorkloadSimulator(
        int maxCpuMs = 70,
        int maxMemMb = 12,
        double cycleSec = 2.5,
        string personality = "business"
    )
    {
        _maxCpuMs = maxCpuMs;
        _maxMemMb = maxMemMb;
        _cycleSec = cycleSec;
        _personality = personality;
    }

    private double Intensity(double elapsed)
    {
        var t = elapsed;
        var baseVal = 0.5 + 0.3 * Math.Sin(2 * Math.PI * t / 300);

        if (_personality == "business")
        {
            var simHour = (t % 600) / 25.0;
            double dayFactor;
            if (simHour >= 8 && simHour <= 18)
                dayFactor = 0.4 + 0.5 * Math.Sin(Math.PI * (simHour - 8) / 10);
            else
                dayFactor = 0.12;
            baseVal = baseVal * 0.15 + dayFactor * 0.85;
        }
        else if (_personality == "spike")
        {
            var roll = _rng.NextDouble();
            var spike = roll < 0.08 ? 0.85 : 0.25;
            baseVal = baseVal * 0.35 + spike * 0.65;
        }
        else if (_personality == "batch")
        {
            var burstPhase = t % 60;
            var burst = burstPhase < 15 ? 0.9 : 0.15;
            baseVal = baseVal * 0.25 + burst * 0.75;
        }

        baseVal *= 0.8 + _rng.NextDouble() * 0.4;
        return Math.Max(0.05, Math.Min(1.0, baseVal));
    }

    private void CpuWork(int ms)
    {
        var end = DateTime.UtcNow.AddMilliseconds(ms);
        while (DateTime.UtcNow < end)
        {
            _ = Math.Sqrt(_rng.NextDouble()) * Math.Sin(_rng.NextDouble());
        }
    }

    private void AdjustMemory(int targetMb)
    {
        var currentMb = _memPool.Count;
        if (targetMb > currentMb)
        {
            var alloc = Math.Min(targetMb - currentMb, 2);
            for (var i = 0; i < alloc; i++)
                _memPool.Add(new byte[1024 * 1024]);
        }
        else if (targetMb < currentMb)
        {
            var release = Math.Max(1, (currentMb - targetMb) / 3);
            _memPool.RemoveRange(0, release);
        }
    }

    private void TouchMemory()
    {
        if (_memPool.Count > 0)
        {
            var idx = _rng.Next(_memPool.Count);
            _memPool[idx][_rng.Next(1024)] = (byte)_rng.Next(256);
        }
    }

    private async Task Run()
    {
        while (_running)
        {
            var elapsed = (DateTime.UtcNow - _startTime).TotalSeconds;
            var intensity = Intensity(elapsed);

            var cpuMs = (int)(intensity * _maxCpuMs);
            CpuWork(cpuMs);

            var targetMb = Math.Max(1, (int)(intensity * _maxMemMb));
            AdjustMemory(targetMb);
            TouchMemory();

            var sleepSec = Math.Max(0.5, _cycleSec + (_rng.NextDouble() - 0.5));
            await Task.Delay(TimeSpan.FromSeconds(sleepSec));
        }
    }

    public void Start()
    {
        Task.Run(Run);
    }

    public void Stop()
    {
        _running = false;
    }
}
