using Microsoft.Extensions.ObjectPool;
using QuestDB;
using QuestDB.Senders;

namespace TelemetryService.Infrastructure.Persistence;

public class QuestDbSenderPool
{
    // Pool of 8 HTTP senders (allows 4 parallel + 4 spare for rotation)
    private static readonly ObjectPool<ISender> _pool = new DefaultObjectPool<ISender>(new QuestDbSenderPooledObjectPolicy(), maximumRetained: 8);

    public static ISender Get() => _pool.Get();
    public static void Return(ISender sender) => _pool.Return(sender);

}

public class QuestDbSenderPooledObjectPolicy : IPooledObjectPolicy<ISender>
{
    public ISender Create()
    {
        var host = Environment.GetEnvironmentVariable("QUESTDB_HTTP_HOST") ?? "questdb";
        var port = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_HTTP_PORT"), out var p) ? p : 9000;

        return Sender.New($"http::addr={host}:{port};auto_flush_rows=10000;request_timeout=60000;");
    }

    public bool Return(ISender obj) => true;
}
