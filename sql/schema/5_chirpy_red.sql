-- +goose Up
ALTER TABLE users
ADD COLUMN is_chirpy_red BOOL DEFAULT FALSE;
