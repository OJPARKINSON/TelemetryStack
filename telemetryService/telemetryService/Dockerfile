# If using Option 2 (port 5000), update your Dockerfile:

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy project files
COPY telemetryService.sln .
COPY src/TelemetryService.API/TelemetryService.API.csproj src/TelemetryService.API/
COPY src/TelemetryService.Application/TelemetryService.Application.csproj src/TelemetryService.Application/
COPY src/TelemetryService.Domain/TelemetryService.Domain.csproj src/TelemetryService.Domain/
COPY src/TelemetryService.Infrastructure/TelemetryService.Infrastructure.csproj src/TelemetryService.Infrastructure/

# Restore dependencies
RUN dotnet restore src/TelemetryService.API/TelemetryService.API.csproj

# Copy source code
COPY . .

# Build and publish
RUN dotnet publish src/TelemetryService.API/TelemetryService.API.csproj \
    -c Release \
    -o /app/publish \
    --no-restore

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:8.0-alpine AS runtime
WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S dotnet && \
    adduser -S dotnet -u 1001 -G dotnet

# Copy published app
COPY --from=build --chown=dotnet:dotnet /app/publish .

USER dotnet

# Configure for port 5000
ENV ASPNETCORE_URLS=http://+:5000 \
    ASPNETCORE_ENVIRONMENT=Production \
    DOTNET_RUNNING_IN_CONTAINER=true \
    DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=1

EXPOSE 5000

ENTRYPOINT ["dotnet", "TelemetryService.API.dll"]