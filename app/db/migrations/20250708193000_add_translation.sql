-- +goose Up
ALTER TABLE german_words ADD COLUMN translation TEXT;
