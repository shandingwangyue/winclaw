#!/usr/bin/env python3
"""
Weather Skill for WinClaw
Uses Open-Meteo API (free, no API key required)
"""

import sys
import json
import urllib.request
import urllib.parse


def get_weather(lat, lon):
    try:
        url = f"https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lon}&current=temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m&daily=weather_code,temperature_2m_max,temperature_2m_min&timezone=auto"

        with urllib.request.urlopen(url, timeout=10) as response:
            data = json.loads(response.read().decode())

        current = data.get("current", {})
        daily = data.get("daily", {})

        weather_codes = {
            0: "Clear sky",
            1: "Mainly clear",
            2: "Partly cloudy",
            3: "Overcast",
            45: "Fog",
            48: "Depositing rime fog",
            51: "Light drizzle",
            53: "Moderate drizzle",
            55: "Dense drizzle",
            61: "Slight rain",
            63: "Moderate rain",
            65: "Heavy rain",
            71: "Slight snow",
            73: "Moderate snow",
            75: "Heavy snow",
            80: "Slight rain showers",
            81: "Moderate rain showers",
            82: "Violent rain showers",
            95: "Thunderstorm",
            96: "Thunderstorm with hail",
        }

        weather_desc = weather_codes.get(current.get("weather_code", 0), "Unknown")

        result = {
            "temperature": current.get("temperature_2m", "N/A"),
            "feels_like": current.get("apparent_temperature", "N/A"),
            "humidity": current.get("relative_humidity_2m", "N/A"),
            "wind_speed": current.get("wind_speed_10m", "N/A"),
            "condition": weather_desc,
            "daily_max": daily.get("temperature_2m_max", ["N/A"])[0],
            "daily_min": daily.get("temperature_2m_min", ["N/A"])[0],
        }
        return result
    except Exception as e:
        return {"error": str(e)}


def geocode(city_name):
    try:
        url = f"https://geocoding-api.open-meteo.com/v1/search?name={urllib.parse.quote(city_name)}&count=1"

        with urllib.request.urlopen(url, timeout=10) as response:
            data = json.loads(response.read().decode())

        if data.get("results"):
            result = data["results"][0]
            return {
                "lat": result.get("latitude"),
                "lon": result.get("longitude"),
                "name": result.get("name"),
                "country": result.get("country", ""),
            }
        return None
    except Exception as e:
        return None


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No parameters provided"}))
        return

    try:
        params = json.loads(sys.argv[1])
    except:
        params = {}

    city = params.get("city", "")
    lat = params.get("latitude")
    lon = params.get("longitude")

    if not lat or not lon:
        if city:
            geo = geocode(city)
            if geo:
                lat, lon = geo["lat"], geo["lon"]
            else:
                print(json.dumps({"error": f"City '{city}' not found"}))
                return
        else:
            print(json.dumps({"error": "Please provide city or coordinates"}))
            return

    weather = get_weather(lat, lon)
    print(json.dumps(weather))


if __name__ == "__main__":
    main()
