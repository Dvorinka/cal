// Open-Meteo weather — no API key, CORS-enabled. Geocode the settings city,
// then pull today's forecast. Cached per-city for 30 min.

export interface Weather {
  temp: number;
  high: number;
  low: number;
  code: number; // WMO weather code
}

const cache = new Map<string, { at: number; weather: Weather }>();
const TTL = 30 * 60 * 1000;

export async function fetchWeather(city: string): Promise<Weather | null> {
  const hit = cache.get(city);
  if (hit && Date.now() - hit.at < TTL) return hit.weather;
  const geo = await fetch(
    `https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(city)}&count=1`
  ).then((r) => r.json());
  const place = geo?.results?.[0];
  if (!place) return null;
  const fc = await fetch(
    `https://api.open-meteo.com/v1/forecast?latitude=${place.latitude}&longitude=${place.longitude}` +
      `&current=temperature_2m,weather_code&daily=temperature_2m_max,temperature_2m_min&timezone=auto&forecast_days=1`
  ).then((r) => r.json());
  const weather: Weather = {
    temp: Math.round(fc.current.temperature_2m),
    high: Math.round(fc.daily.temperature_2m_max[0]),
    low: Math.round(fc.daily.temperature_2m_min[0]),
    code: fc.current.weather_code,
  };
  cache.set(city, { at: Date.now(), weather });
  return weather;
}

// WMO weather codes → Lucide icon name + label.
export function weatherIcon(code: number): { icon: string; label: string } {
  if (code === 0) return { icon: "sun", label: "Clear" };
  if (code <= 3) return { icon: "cloud-sun", label: "Partly cloudy" };
  if (code <= 48) return { icon: "cloud-fog", label: "Fog" };
  if (code <= 55) return { icon: "cloud-drizzle", label: "Drizzle" };
  if (code <= 57 || (code >= 61 && code <= 65)) return { icon: "cloud-rain", label: "Rain" };
  if (code <= 77) return { icon: "snowflake", label: "Snow" };
  if (code >= 80 && code <= 82) return { icon: "cloud-rain", label: "Showers" };
  if (code >= 85 && code <= 86) return { icon: "snowflake", label: "Snow" };
  if (code >= 95) return { icon: "cloud-lightning", label: "Storm" };
  return { icon: "cloud", label: "Cloudy" };
}
