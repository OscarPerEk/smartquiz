-- +goose Up
create table if not exists german_words(
	id integer primary key,
	example text unique not null,
	german_word text not null,
	definition text not null,
	created_at datetime not null,
	updated_at datetime not null,
	deleted_at datetime
);

-- +goose Down
drop table if exists users;
