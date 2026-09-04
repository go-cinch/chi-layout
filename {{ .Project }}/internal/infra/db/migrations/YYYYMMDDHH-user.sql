-- +migrate Up
{{ if eq .Computed.database_driver_final "mysql" }}
CREATE TABLE t_user (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
{{ else }}
CREATE TABLE t_user (
    id BIGSERIAL PRIMARY KEY,
{{ end }}
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    CONSTRAINT uk_user_email UNIQUE (email)
);

-- +migrate Down
DROP TABLE t_user;
