using TelemetryService.Infrastructure.Configuration;
namespace TelemetryService.InfrastructureTests;

public class DotEnvTests
{
    private string _testDir = string.Empty;
    private readonly Dictionary<string, string> _originalEnvVars = new();

    [SetUp]
    public void Setup()
    {
        _testDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(_testDir);

        CaptureOriginalEnvVar();
    }

    [TearDown]
    public void TearDown()
    {
        if (Directory.Exists(_testDir))
        {
            Directory.Delete(_testDir, true);

            RestoreOriginalEnvironmentVariables();
        }
    }

    [Test]
    public void Load_WhenFileDoesNotExist_DoesNotThrow()
    {
        var nonExistentPath = Path.Combine(_testDir, "nonExistant.env");

        Assert.DoesNotThrow(() => DotEnv.Load(nonExistentPath));
    }

    [Test]
    public void Load_WhenLineHasNoEquals_SkipLine()
    {
        var envFilePath = CreateTestEnvFile("INVALID_LINE_NO_EQUALS\nVALID_VAR=valid_value");

        DotEnv.Load(envFilePath);

        Assert.That(Environment.GetEnvironmentVariable("INVALID_LINE_NO_EQUALS"), Is.Null);
        Assert.That(Environment.GetEnvironmentVariable("VALID_VAR"), Is.EqualTo("valid_value"));
    }

    private string CreateTestEnvFile(string envString)
    {
        var filePath = Path.Combine(_testDir, $"{Guid.NewGuid()}.env");
        File.WriteAllText(filePath, envString);
        return filePath;
    }

    private void CaptureOriginalEnvVar()
    {
        var testVars = new[] { "TEST_VAR", "ANOTHER_VAR", "VALID_VAR", "EXISTING_VAR", "EMPTY_VAR", "VAR_WITH_EQUALS", "FIRST_VAR", "SECOND_VAR" };

        foreach (var variable in testVars)
        {
            var value = Environment.GetEnvironmentVariable(variable);
            if (value != null)
            {
                _originalEnvVars[variable] = value;
            }
        }
    }

    private void RestoreOriginalEnvironmentVariables()
    {
        var testVariables = new[] { "TEST_VAR", "ANOTHER_VAR", "VALID_VAR", "EXISTING_VAR", "EMPTY_VAR", "VAR_WITH_EQUALS", "FIRST_VAR", "SECOND_VAR" };

        foreach (var variable in testVariables)
        {
            if (_originalEnvVars.TryGetValue(variable, out var originalValue))
            {
                Environment.SetEnvironmentVariable(variable, originalValue);
            }
            else
            {
                Environment.SetEnvironmentVariable(variable, null);
            }
        }
    }
}
