CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "email" varchar UNIQUE NOT NULL,
  "name" varchar,
  "password" varchar,
  "created_at" timestamptz NOT NULL DEFAULT 'now()'
);

CREATE TABLE "goals" (
  "id" bigserial PRIMARY KEY,
  "title" varchar NOT NULL,
  "description" text,
  "target_amount" bigint,
  "collected_amount" bigint NOT NULL DEFAULT 0,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT 'now()'
);

CREATE TABLE "donations" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint,
  "goal_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "currency" varchar NOT NULL DEFAULT 'USD',
  "is_anonymous" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT 'now()'
);

CREATE INDEX ON "users" ("email");

CREATE INDEX ON "donations" ("goal_id");

CREATE INDEX ON "donations" ("user_id");

COMMENT ON COLUMN "goals"."target_amount" IS 'in smallest currency unit, e.g., cents';

COMMENT ON COLUMN "donations"."amount" IS 'must be positive';

ALTER TABLE "donations" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "donations" ADD FOREIGN KEY ("goal_id") REFERENCES "goals" ("id");
