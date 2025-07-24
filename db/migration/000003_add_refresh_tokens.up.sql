CREATE TABLE "refresh_tokens" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id" bigint NOT NULL,
  "token_id" uuid NOT NULL UNIQUE,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "revoked_at" timestamptz
);

ALTER TABLE "refresh_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

CREATE INDEX ON "refresh_tokens" ("user_id");
CREATE INDEX ON "refresh_tokens" ("token_id");
CREATE INDEX ON "refresh_tokens" ("expires_at");
