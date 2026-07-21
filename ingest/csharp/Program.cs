using Ingest.Cli;
using Ingest.Ibt;
using Ingest.Ibt.Headers;

var opts = ProcessCommand.Parse(args);

var ibtFiles = Directory.GetFiles(opts.Path);

foreach (string ibtRelPath in ibtFiles)
{
    Console.WriteLine($"Filename: {ibtRelPath}");
    using IbtFile file = new(ibtRelPath);
    IbtHeaders headers = IbtHeaders.Parse(file.Span);
    TickParser parser = new(headers);

    var tick = new TelemetryTick();
    while (parser.Next(ref tick, file.Span))
    {
        Console.WriteLine($"parser {parser.ToString()}");
    }


}