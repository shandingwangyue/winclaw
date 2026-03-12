---
name: weather
description: Fetches current weather and forecast for any city using Open-Meteo API. Use when user asks about weather, temperature, or forecast.
---

# Weather Skill

Fetches current weather and forecast for any city using Open-Meteo API (free, no API key required).

## Parameters

- `city` (string, optional): City name (e.g., "Beijing", "Tokyo")
- `latitude` (number, optional): Latitude coordinate
- `longitude` (number, optional): Longitude coordinate

## Returns

Current temperature, feels-like temperature, humidity, wind speed, weather condition, and daily forecast.

## Example

```json
{"city": "Beijing"}
{"latitude": 39.9, "longitude": 116.4}
```
