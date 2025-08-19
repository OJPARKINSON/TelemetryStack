# Certificate Setup for Traefik + Tailscale Integration

This document explains the certificate configuration for secure HTTPS access to your IRacing Display services.

## Overview

The setup uses **Tailscale Certificates** for automatic Let's Encrypt certificates on Tailscale domains (`*.ts.net`).

## Prerequisites

### Tailscale Setup
1. Enable MagicDNS in your Tailscale admin console
2. Enable HTTPS certificates in the DNS settings
3. Ensure Tailscale daemon is running on the host machine

## Configuration

### Environment Variables
Set your Tailscale domain in `.env`:
```bash
TAILSCALE_DOMAIN=your-device-name.tailxxxxx.ts.net
LOCAL_DOMAIN=192.168.1.202
```

### Access URLs

**Via Tailscale (Trusted Certificates)**
- Dashboard: `https://${TAILSCALE_DOMAIN}/dashboard/`
- Grafana: `https://${TAILSCALE_DOMAIN}/grafana/`
- RabbitMQ: `https://${TAILSCALE_DOMAIN}/rabbitmq/`
- Prometheus: `https://${TAILSCALE_DOMAIN}/prometheus/`
- QuestDB: `https://${TAILSCALE_DOMAIN}/questdb/`

**Via Local Network (No HTTPS)**
- Use HTTP on local domain for development: `http://${LOCAL_DOMAIN}:8080`

## Certificate Management

- **Automatically renewed** by Traefik 14 days before expiration
- **No manual intervention** required
- **Trusted by all browsers** on your Tailscale network

## Security Features

- TLS 1.2+ enforced with strong cipher suites
- HSTS headers enabled
- Security headers middleware applied
- Rate limiting configured per service

## Troubleshooting

1. **Check Tailscale daemon**: `sudo systemctl status tailscaled`
2. **Verify connection**: `tailscale status`
3. **Validate config**: `docker-compose config`
4. **Check router status**: Visit `http://localhost:8080` (Traefik dashboard)