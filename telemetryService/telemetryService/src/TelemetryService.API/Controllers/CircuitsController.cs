using Microsoft.AspNetCore.Mvc;
using System.Text.Json;

namespace TelemetryService.API.Controllers;

[Route("api/[controller]")]
[ApiController]
public class CircuitsController : ControllerBase
{
    private readonly IWebHostEnvironment _environment;

    public CircuitsController(IWebHostEnvironment environment)
    {
        _environment = environment;
    }

    [HttpGet]
    public async Task<IActionResult> GetCircuit()
    {
        var filePath = Path.Combine(_environment.ContentRootPath, "Data", "tracks.json");

        if (!System.IO.File.Exists(filePath))
        {
            return NotFound(new { error = "Track file not found" });
        }

        var jsonString = await System.IO.File.ReadAllTextAsync(filePath);

        var jsonData = JsonSerializer.Deserialize<object>(jsonString);

        return Ok(jsonData);
    }
}