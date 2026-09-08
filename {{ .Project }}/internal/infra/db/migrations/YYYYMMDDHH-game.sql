-- +migrate Up
{{ if eq .Computed.database_driver_final "mysql" }}
CREATE TABLE t_game (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
{{ else }}
CREATE TABLE t_game (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
{{ end }}
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    CONSTRAINT ck_game_name CHECK (CHAR_LENGTH(TRIM(name)) > 0)
);

-- +migrate Down
DROP TABLE t_game;
