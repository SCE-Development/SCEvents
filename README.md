# SCEvents
Welcome to SCE's event planner platform!

## Local Setup (requires Docker)

### 1. Clone the repository
```
git clone https://github.com/SCE-Development/SCEvents.git
cd SCEvents
```
### 2. Start the services
```
docker compose -f docker-compose.dev.yml up --build -d
```

NOTE: To test SCEvents together with the frontend, you'll need to run Clark separately [following the instructions here](https://github.com/SCE-Development/Clark/wiki/Getting-Started)

### 3. Point Clark at your local SCEvents server

If you want Clark to talk to a locally running SCEvents instance, update Clark's config so the SCEvents base URL points to port `8002`.

In `Clark/src/config/config.json`:

```json
{
  "SCEvents": {
    "BASE_URL": "http://localhost:8002"
  }
}
```

This is the local-development setting for running Clark and SCEvents as separate processes on your machine.

For deployed environments behind Clark's nginx proxy, use:

```json
{
  "SCEvents": {
    "BASE_URL": "/api/scevents"
  }
}
```

## Testing

For integration tests, load testing, and related Make targets, see [testing/README.md](testing/README.md).

## Tech Stack
- [SCEvents Documentation](https://docs.google.com/document/d/1K1va_5XcoPiI-ZY19f7KfWbIexqOdnR_RKz9MlP3C2U)
- [Excalidraw Diagrams](https://excalidraw.com/#json=blzinrcwsw_ZrNasTCHi5,2kjWQhJKEaD1ROrF5Nan3A)

![SCEvents Diagram](scevents.png)

