using Microsoft.AspNetCore.Mvc;
using TelemetryService.Infrastructure.Persistence;

namespace TelemetryService.API.Controllers;

[Route("api/[controller]")]
[ApiController]
public class HealthController : ControllerBase
{
    public HealthController()
    {
        // QuestDbService is now created per-request for thread safety
    }

    [HttpGet]
    public IActionResult GetTest()
    {
        return Ok(new { message = "Test endpoint working!", timestamp = DateTime.UtcNow });
    }
}
