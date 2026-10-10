"""NOAA reader and deterministic geography/time checks for a local experiment."""
from datetime import datetime, timezone
import requests

NOAA_URL = "https://api.weather.gov/alerts/active?area=FL"
FACILITY = {"id": "FL-DEMO-01", "name": "Big Bend Field Office", "latitude": 29.99, "longitude": -83.44, "fictional": True}

def parse_time(value):
    if not value:
        return None
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        return parsed if parsed.tzinfo else None
    except (ValueError, TypeError):
        return None

def ring_contains(point, ring):
    """Ray casting with boundary included; coordinates are longitude, latitude."""
    x, y = point
    inside = False
    for (ax, ay), (bx, by) in zip(ring, ring[1:] + ring[:1]):
        cross = (x-ax)*(by-ay) - (y-ay)*(bx-ax)
        if abs(cross) < 1e-10 and min(ax,bx) <= x <= max(ax,bx) and min(ay,by) <= y <= max(ay,by):
            return True
        if (ay > y) != (by > y) and x < (bx-ax)*(y-ay)/(by-ay) + ax:
            inside = not inside
    return inside

def geography(geometry, facility=FACILITY):
    if not geometry:
        return "unknown"
    try:
        polygons = [geometry["coordinates"]] if geometry["type"] == "Polygon" else geometry["coordinates"] if geometry["type"] == "MultiPolygon" else None
        if not polygons or any(not polygon or any(len(ring)<4 for ring in polygon) for polygon in polygons):
            return "unknown"
        point = [facility["longitude"], facility["latitude"]]
        covered = any(ring_contains(point, p[0]) and not any(ring_contains(point, hole) for hole in p[1:]) for p in polygons)
        return "applicable" if covered else "outside"
    except (KeyError, TypeError, ValueError):
        return "unknown"

def normalize(feature, now, facility=FACILITY):
    p = feature.get("properties", {})
    start, end = parse_time(p.get("effective")), parse_time(p.get("expires"))
    lifecycle = "cancelled" if p.get("messageType") == "Cancel" else "unknown" if start is None or end is None else "not-yet-effective" if now < start else "expired" if now >= end else "active"
    geo = geography(feature.get("geometry"), facility)
    applicable = "unknown" if lifecycle=="unknown" or geo=="unknown" else "applicable" if lifecycle=="active" and geo=="applicable" and p.get("status")=="Actual" else "not-applicable"
    return {"id":p.get("id") or feature.get("id"), "source":feature.get("id"), "issuer":p.get("senderName"), "event":p.get("event"), "headline":p.get("headline"), "description":p.get("description"), "instruction":p.get("instruction"), "area":p.get("areaDesc"), "issued":p.get("sent"), "effective":p.get("effective"), "expires":p.get("expires"), "severity":p.get("severity"), "certainty":p.get("certainty"), "urgency":p.get("urgency"), "status":p.get("status"), "messageType":p.get("messageType"), "references":p.get("references",[]), "geography":geo, "lifecycle":lifecycle, "applicability":applicable}

def fetch_evidence():
    response = requests.get(NOAA_URL, headers={"User-Agent":"WeatherDesk-HackMidwest-personal-experiment", "Accept":"application/geo+json"}, timeout=(5,20))
    response.raise_for_status()
    payload = response.json()
    features = payload.get("features")
    if not isinstance(features,list):
        raise ValueError("NOAA response has no alert collection")
    now = datetime.now(timezone.utc)
    return {"mode":"LIVE", "facility":FACILITY, "lastSuccessfulRefresh":now.isoformat(), "feedUpdated":payload.get("updated"), "source":NOAA_URL, "alerts":[normalize(f,now) for f in features], "gaps":["Alerts without geometry remain unknown; zone lookup is not implemented in this experiment.", "No employee locations or commute routes are known.", "No supplementary road or power signal is connected.", "No reopening decision follows from absent, expired, or cancelled alerts."]}
