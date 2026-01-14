# WhatsApp Recovery System - Local Development Environment

Complete Docker-based local testing environment for the Automated WhatsApp Recovery System.

## Quick Start

```bash
# Navigate to docker directory
cd docker

# Make scripts executable (Linux/Mac)
chmod +x ../local/scripts/*.sh

# Run setup script
../local/scripts/setup.sh

# View logs
docker-compose logs -f
```

## Architecture

The local environment consists of:

- **Event Tracker Lambda** - HTTP API on port 8080
- **Worker Lambda** - Scheduled service (5-minute interval)
- **Redis** - State and recovery history storage
- **AppConfig Mock** - File-based configuration server (port 2772)
- **WhatsApp Mock** - Mock API for message logging (port 8081)

## Service Endpoints

| Service | URL | Description |
|---------|-----|-------------|
| Event Tracker API | http://localhost:8080 | POST journey events |
| AppConfig Mock | http://localhost:2772 | Journey configurations |
| WhatsApp Mock | http://localhost:8081 | Mock WhatsApp API |
| Redis | localhost:6379 | State storage |

## Environment Configuration

### Event Tracker (.env.event-tracker)

```env
REDIS_ADDR=redis:6379          # Redis connection
STATE_TTL=24h                  # Journey state TTL
DEBUG=true                     # Enable debug logs
```

### Worker (.env.worker)

```env
REDIS_ADDR=redis:6379                       # Redis connection
APPCONFIG_ENDPOINT=http://appconfig-mock:2772  # Config server
WHATSAPP_API_ENDPOINT=http://whatsapp-mock:8081  # WhatsApp API
WHATSAPP_MOCK=true                          # Enable mock mode
WORKER_INTERVAL=5m                          # Execution interval
DEBUG=true                                  # Enable debug logs
```

## Journey Configuration

Journey configurations are stored in `local/configs/`:

- `journey.onboarding-v2.yaml` - Recovery rules per step
- `journey.onboarding-v2.templates.yaml` - Message templates

### Example Journey Config

```yaml
journey: onboarding-v2
global:
  enabled: true
  max_total_attempts: 5
  min_interval_between_attempts_minutes: 15
steps:
  - name: personal-data
    recovery_rules:
      - name: early-reminder
        enabled: true
        inactive_minutes: 10
        max_attempts: 1
        template: personal-data-soft
```

## Testing Workflow

### 1. Start Environment

```bash
cd docker
docker-compose up -d
```

### 2. Post a Journey Event

```bash
curl -X POST http://localhost:8080/journey/event \
  -H "Content-Type: application/json" \
  -d '{
    "journey_id": "onboarding-v2",
    "step": "personal-data",
    "customer_number": "5511999999999",
    "tenant_id": "tenant-123",
    "contact_id": "contact-456",
    "metadata": {
      "nome_cliente": "João Silva",
      "link": "https://example.com/continue"
    }
  }'
```

### 3. Inspect Redis State

```bash
docker-compose exec redis redis-cli

> KEYS journey:*
> GET journey:onboarding-v2:5511999999999:state
> LRANGE journey:onboarding-v2:5511999999999:repiques 0 -1
```

### 4. View Worker Logs

```bash
docker-compose logs -f worker
```

Look for:
- "Starting scheduled worker execution"
- "Loaded journey config"
- "MOCK: Would send WhatsApp message" (when recovery triggers)

### 5. Complete a Journey

```bash
curl -X POST http://localhost:8080/journey/finish \
  -H "Content-Type: application/json" \
  -d '{
    "journey_id": "onboarding-v2",
    "customer_number": "5511999999999"
  }'
```

## Helper Scripts

### Setup Script

```bash
./local/scripts/setup.sh
```

Builds and starts all services with health checks.

### Test API Script

```bash
# Interactive mode
./local/scripts/test-api.sh

# Command line mode
./local/scripts/test-api.sh post                    # Post default event
./local/scripts/test-api.sh finish                  # Finish journey
./local/scripts/test-api.sh state                   # Show state
./local/scripts/test-api.sh history                 # Show history
./local/scripts/test-api.sh trigger                 # Restart worker
./local/scripts/test-api.sh clear                   # Clear all data
```

### Redis Inspector Script

```bash
# Interactive mode
./local/scripts/inspect-redis.sh

# Command line mode
./local/scripts/inspect-redis.sh states             # List all states
./local/scripts/inspect-redis.sh histories          # List all histories
./local/scripts/inspect-redis.sh customer           # Show customer
./local/scripts/inspect-redis.sh stats              # Show statistics
./local/scripts/inspect-redis.sh cli                # Open Redis CLI
```

## Common Commands

### View All Logs

```bash
docker-compose logs -f
```

### View Specific Service Logs

```bash
docker-compose logs -f event-tracker
docker-compose logs -f worker
docker-compose logs -f appconfig-mock
docker-compose logs -f whatsapp-mock
```

### Restart a Service

```bash
docker-compose restart worker
```

### Stop All Services

```bash
docker-compose down
```

### Clean Slate (Remove All Data)

```bash
docker-compose down -v
docker-compose up -d
```

### Rebuild After Code Changes

```bash
docker-compose build
docker-compose up -d
```

## Testing Recovery Flow

### Test Immediate Recovery (10 minutes)

1. Post an event for a customer
2. Wait 10 minutes OR manually trigger worker
3. Check worker logs for "early-reminder" execution
4. Inspect Redis for recovery history

### Test Progressive Recovery (30 minutes)

1. Post an event for a customer
2. Wait 10 minutes - "early-reminder" triggers (1 attempt)
3. Wait additional 20 minutes (30 total) - "late-reminder" triggers
4. Check history shows 2 separate recovery attempts with different templates

### Test Global Limits

1. Configure `max_total_attempts: 3` in journey config
2. Create multiple recovery rules
3. Verify no more than 3 total messages are sent

### Test Journey Completion

1. Post an event for a customer
2. Call finish endpoint before recovery triggers
3. Verify worker skips the customer (state deleted from Redis)

## Configuration Hot-Reload

The system supports configuration hot-reload:

1. Edit `local/configs/journey.onboarding-v2.yaml`
2. Worker picks up changes on next execution (no restart needed)
3. Verify changes in worker logs

## Troubleshooting

### Worker Not Processing

**Check Redis connection:**
```bash
docker-compose logs redis
docker-compose exec redis redis-cli ping
```

**Check AppConfig mock:**
```bash
docker-compose logs appconfig-mock
curl http://localhost:2772/journey.onboarding-v2.yaml
```

**Check worker logs:**
```bash
docker-compose logs worker | grep -i error
```

### Event Tracker Not Responding

**Test health endpoint:**
```bash
curl http://localhost:8080/health
```

**Check port binding:**
```bash
docker-compose ps
```

**View logs:**
```bash
docker-compose logs event-tracker
```

### Configs Not Loading

**Verify file names match pattern:**
```bash
ls -la local/configs/
# Should see: journey.{journey-id}.yaml
```

**Check AppConfig mock logs:**
```bash
docker-compose logs appconfig-mock
# Look for 404 errors
```

**Verify volume mount:**
```bash
docker-compose config | grep configs
```

### Messages Not Sending

**Check mock mode enabled:**
```bash
docker-compose exec worker env | grep WHATSAPP_MOCK
# Should show: WHATSAPP_MOCK=true
```

**Check worker logs:**
```bash
docker-compose logs worker | grep "MOCK:"
```

## Development Workflow

### Modify Journey Rules

1. Edit `local/configs/journey.onboarding-v2.yaml`
2. Change `inactive_minutes`, `max_attempts`, or other settings
3. Worker picks up changes automatically

### Modify Templates

1. Edit `local/configs/journey.onboarding-v2.templates.yaml`
2. Update message text and variable interpolation
3. Changes apply on next worker execution

### Modify Code

1. Make code changes in `event-tracker/` or `worker/`
2. Rebuild containers: `docker-compose build`
3. Restart services: `docker-compose up -d`

### Debug Single Customer

1. Post event for test customer
2. Check state: `docker-compose exec redis redis-cli GET journey:onboarding-v2:5511999999999:state`
3. Run worker with DEBUG=true (already enabled)
4. View detailed logs

## Data Persistence

- **Redis data** persists in named volume `redis-data`
- **Configs** mounted read-only from `local/configs/`
- **Env files** in `local/.env.*`

To clear all data:
```bash
docker-compose down -v  # Deletes volumes
```

## Production vs Local

| Aspect | Production | Local |
|--------|-----------|-------|
| Event Tracker | API Gateway + Lambda | HTTP server in Docker |
| Worker | EventBridge schedule | Loop with 5m interval |
| Redis | ElastiCache | Docker container |
| AppConfig | AWS AppConfig | File-based mock |
| WhatsApp | Business API | Mock server |
| Logs | CloudWatch | Console/Docker logs |

## Migration to Production

When ready for production:

1. Environment variables already match AWS Lambda format
2. Code detects Lambda via `AWS_LAMBDA_FUNCTION_NAME`
3. Update `.env` files with production values:
   - `APPCONFIG_ENDPOINT` → AWS AppConfig endpoint
   - `REDIS_ADDR` → ElastiCache endpoint
   - `WHATSAPP_MOCK=false` with real credentials
4. Deploy using existing Dockerfile
5. No code changes required

## Performance Notes

- Worker scans Redis every 5 minutes by default
- Configurable via `WORKER_INTERVAL` environment variable
- Redis uses pipelining for efficient batch operations
- AppConfig responses are cached to reduce API calls

## Security Notes

- Mock mode prevents actual WhatsApp API calls
- Credentials in `.env` files (never commit to git)
- Read-only config mounts for safety
- Redis password optional (not needed for local testing)

## Support

For issues or questions:
1. Check logs: `docker-compose logs -f [service]`
2. Verify health: `docker-compose ps`
3. Review this README
4. Inspect Redis data with helper scripts
