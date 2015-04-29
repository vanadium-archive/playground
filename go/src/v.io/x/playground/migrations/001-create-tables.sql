-- +migrate Up

CREATE TABLE bundle_data (
	hash BINARY(32) NOT NULL PRIMARY KEY,
	json MEDIUMTEXT NOT NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

CREATE TABLE bundle_link (
	id CHAR(64) CHARACTER SET ascii NOT NULL PRIMARY KEY,
	hash BINARY(32),
	created_at TIMESTAMP NOT NULL DEFAULT now(),
	CONSTRAINT hash_link_to_data FOREIGN KEY (hash) REFERENCES bundle_data(hash) ON DELETE SET NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- +migrate Down

DROP TABLE bundle_link;

DROP TABLE bundle_data;
