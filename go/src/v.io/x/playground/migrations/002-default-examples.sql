-- +migrate Up

ALTER TABLE bundle_link
  ADD COLUMN slug VARCHAR(128) NULL DEFAULT NULL AFTER id,
  ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT false AFTER slug,
  ADD UNIQUE INDEX slug_index (slug),
  ADD INDEX is_default_index (is_default);

-- +migrate Down

ALTER TABLE bundle_link
  DROP INDEX is_default_index,
  DROP INDEX slug_index,
  DROP COLUMN is_default,
  DROP COLUMN slug;
