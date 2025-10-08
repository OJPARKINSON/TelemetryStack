using Microsoft.Extensions.ObjectPool;
using QuestDB;
using QuestDB.Senders;

namespace TelemetryService.Infrastructure.Persistance;

public class QuestDbSenderPool
{
    private static readonly ObjectPool<ISender> _pool = new DefaultObjectPool<ISender>(new QuestDbSenderPooledObjectPolicy(), maximumRetained: 60);

    public static ISender Get() => _pool.Get();
    public static void Return(ISender sender) => _pool.Return(sender);

}

public class QuestDbSenderPooledObjectPolicy : IPooledObjectPolicy<ISender>
{
    public ISender Create()
    {
        var host = Environment.GetEnvironmentVariable("QUESTDB_TCP_HOST") ?? "questdb";
        var port = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_TCP_PORT"), out var p) ? p : 9009;
        return Sender.New($"tcp::addr={host}:{port};auto_flush_rows=1000;auto_flush_interval=1000;");
    }

    public bool Return(ISender obj) => true;
}
