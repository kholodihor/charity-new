CREATE TABLE "events" (
  "id" bigserial PRIMARY KEY,
  "name" varchar NOT NULL,
  "place" varchar NOT NULL,
  "date" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT 'now()'
);

CREATE TABLE "event_bookings" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "event_id" bigint NOT NULL,
  "booked_at" timestamptz NOT NULL DEFAULT 'now()',
  UNIQUE(user_id, event_id)
);

CREATE INDEX ON "events" ("date");
CREATE INDEX ON "events" ("name");
CREATE INDEX ON "event_bookings" ("user_id");
CREATE INDEX ON "event_bookings" ("event_id");

ALTER TABLE "event_bookings" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "event_bookings" ADD FOREIGN KEY ("event_id") REFERENCES "events" ("id");

COMMENT ON COLUMN "events"."date" IS 'event date and time';
COMMENT ON TABLE "event_bookings" IS 'tracks which users have booked which events';
