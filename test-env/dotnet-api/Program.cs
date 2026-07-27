using dotnet_api;

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.UseUrls("http://0.0.0.0:8080");
var app = builder.Build();

var _workload = new WorkloadSimulator(
    maxCpuMs: 70,
    maxMemMb: 12,
    personality: "business"
);
_workload.Start();

app.MapGet("/", () => new { service = "dotnet-api", status = "running", time = DateTime.UtcNow });

app.MapGet("/cpu", (int loops = 1000) =>
{
    var result = 0.0;
    for (var i = 0; i < loops; i++)
        result += Math.Sqrt(i) * Math.Sin(i);
    return new { service = "dotnet-api", result, loops };
});

app.MapGet("/memory", (int mb = 10) =>
{
    var data = new List<byte[]>();
    for (var i = 0; i < mb; i++)
        data.Add(new byte[1024 * 1024]);
    GC.Collect();
    return new { service = "dotnet-api", allocatedMB = mb };
});

app.MapGet("/health", () => new { status = "healthy", ts = DateTime.UtcNow });

app.Run();
