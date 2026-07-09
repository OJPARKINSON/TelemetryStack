using Ingest.Cli;
using Ingest.Ibt;
using Ingest.Ibt.Headers;

var opts = ProcessCommand.Parse(args);

var ibtFiles = Directory.GetFiles(opts.Path);

foreach (string ibt in ibtFiles)
{
    Console.WriteLine($"{ibt}");
    using IbtFile file = new(ibt);
    IbtHeaders headers = IbtHeaders.Parse(file.Span);
    TickParser parser = new(headers);

    Console.WriteLine(headers);
}