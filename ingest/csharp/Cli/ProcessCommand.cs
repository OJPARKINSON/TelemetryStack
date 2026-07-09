using System.CommandLine;
using System.Runtime.CompilerServices;

namespace Ingest.Cli;

public readonly record struct ProcessOptions(string Path);

public static class ProcessCommand
{
    private static readonly Option<string> PathOption = new("--path", "-p")
    {
        Description = "Path to the IRacing Telemetry folder",
        DefaultValueFactory = _ => "../ibt_files/" // For me in dev
    };

    public static ProcessOptions Parse(string[] args)
    {
        var rootCmd = new RootCommand("process");
        rootCmd.Options.Add(PathOption);

        var result = rootCmd.Parse(args);
        return new ProcessOptions(result.GetValue(PathOption)!);
    }
}
