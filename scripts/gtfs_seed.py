import csv
import os
import psycopg2
from collections import defaultdict

GTFS_DIR = "temp/gtfs"
DB_URL = os.environ["DATABASE_URL"]

def parse_time_secs(t):
    if not t:
        return None
    parts = t.split(":")
    if len(parts) != 3:
        return None
    try:
        return int(parts[0]) * 3600 + int(parts[1]) * 60 + int(parts[2])
    except ValueError:
        return None

def load_calendar():
    cal = {}
    with open(f"{GTFS_DIR}/calendar.txt", encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            days = []
            for i, k in enumerate(["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"]):
                if row.get(k) == "1":
                    days.append(i)
            cal[row["service_id"]] = days
    return cal

def load_trips(calendar):
    trips = {}
    with open(f"{GTFS_DIR}/trips.txt", encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            route_id = row["route_id"]
            parts = route_id.rsplit("_", 1)
            if len(parts) != 2:
                continue
            sub_route_uid, dir_str = parts[0], parts[1]
            try:
                direction = int(dir_str)
            except ValueError:
                continue
            if direction not in (0, 1):
                continue
            service_id = row["service_id"]
            days = calendar.get(service_id, [])
            trips[row["trip_id"]] = {
                "sub_route_uid": sub_route_uid,
                "direction": direction,
                "days": days,
            }
    return trips

def main():
    calendar = load_calendar()
    trips = load_trips(calendar)

    samples = defaultdict(list)

    with open(f"{GTFS_DIR}/stop_times.txt", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        current_trip = None
        origin_secs = None
        for row in reader:
            trip_id = row["trip_id"]
            trip = trips.get(trip_id)
            if not trip:
                continue

            seq = int(row["stop_sequence"])
            arr_secs = parse_time_secs(row["arrival_time"])

            if seq == 1 or trip_id != current_trip:
                current_trip = trip_id
                origin_secs = arr_secs
                continue

            if arr_secs is None or origin_secs is None:
                continue

            travel_secs = arr_secs - origin_secs
            if travel_secs <= 0 or travel_secs > 7200:
                continue

            stop_uid = row["stop_id"]
            dep_hour = origin_secs // 3600
            for dow in trip["days"]:
                key = (trip["sub_route_uid"], trip["direction"], stop_uid, dep_hour % 24, dow)
                samples[key].append(travel_secs)

    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()
    inserted = 0
    for (sub_route_uid, direction, stop_uid, hour, dow), vals in samples.items():
        median = sorted(vals)[len(vals) // 2]
        cur.execute("""
            INSERT INTO bus_travel_avg
              (sub_route_uid, direction, stop_uid, hour, day_of_week, avg_seconds, sample_count, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, 0, NOW())
            ON CONFLICT (sub_route_uid, direction, stop_uid, hour, day_of_week) DO NOTHING
        """, (sub_route_uid, direction, stop_uid, hour, dow, median))
        inserted += 1
        if inserted % 10000 == 0:
            conn.commit()
            print(f"  {inserted} rows...")
    conn.commit()
    cur.close()
    conn.close()
    print(f"Done: {inserted} rows inserted (sample_count=0)")

if __name__ == "__main__":
    main()
