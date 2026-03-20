# Next.js Telemetry Dashboard Setup

This document explains how to set up and run the Next.js telemetry dashboard that connects to InfluxDB.

## Project Structure

Ensure your project has the following structure:

```
/
├── app/
│   ├── api/
│   │   ├── health/
│   │   │   └── route.ts
│   │   ├── laps/
│   │   │   └── route.ts
│   │   ├── sessions/
│   │   │   └── route.ts
│   │   └── telemetry/
│   │       └── route.ts
│   └── page.tsx
├── lib/
│   └── influxdb.ts
├── .env.influxdb-admin-token
├── .env.influxdb-admin-username
├── .env.influxdb-admin-password
├── Dockerfile
├── docker-compose.yml
├── next.config.js
├── package.json
└── tsconfig.json
```

## Steps to Set Up

1. **Install Dependencies**:

   ```bash
   npm install
   ```

2. **Set Environment Variables**:

   - Create the necessary secret files:
     - `.env.influxdb-admin-token`
     - `.env.influxdb-admin-username`
     - `.env.influxdb-admin-password`
   - Or update `docker-compose.yml` to use different secret sources

3. **Build and Run with Docker**:

   ```bash
   docker-compose up -d
   ```

4. **Access the Dashboard**:
   - Open your browser to `http://localhost:3000`

## Troubleshooting

### Module Resolution Issues

If you encounter module resolution errors:

1. Make sure your `tsconfig.json` has the correct paths configuration:

   ```json
   "paths": {
     "@/*": ["./*"]
   }
   ```

2. Verify that the import paths in API routes are correct:

   ```typescript
   import { getInfluxDBClient, influxConfig } from "../../../lib/influxdb";
   ```

3. Try restarting the Next.js development server or rebuilding the Docker container.

### InfluxDB Connection Issues

If the dashboard can't connect to InfluxDB:

1. Check that InfluxDB is running:

   ```bash
   docker-compose ps
   ```

2. Verify that the InfluxDB credentials are correct
3. Ensure that the bucket and organization exist in your InfluxDB instance
4. Check the network connectivity between containers

## Customizing the Dashboard

To add more features to the dashboard:

1. Add new API endpoints in the `/app/api/` directory
2. Update the UI components in `page.tsx`
3. Add new visualization components as needed

Refer to the Next.js and InfluxDB documentation for more advanced customizations.
