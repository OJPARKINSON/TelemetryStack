using System.CommandLine;

var rootCmd = new RootCommand("process");

var telemetryDirPath = new Option<string>("--path", "-p")
{
    Description = "Path to the IRacing Telemetry folder"
};

Console.WriteLine("Welcome to telemetry stack ingest service!");
