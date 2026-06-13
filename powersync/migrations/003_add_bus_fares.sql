CREATE TABLE IF NOT EXISTS bus_fares (
    sub_route_uid     text PRIMARY KEY,
    fare_pricing_type smallint NOT NULL DEFAULT 0,
    is_free_bus       boolean NOT NULL DEFAULT false,
    section_fares     jsonb,
    updated_at        timestamptz NOT NULL DEFAULT now()
);
