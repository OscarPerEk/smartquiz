-- +goose Up
INSERT INTO german_words (example, german_word, definition, created_at, updated_at)
VALUES
  ('Am Wochenende machen wir ein Picknick im Park.', 'Picknick', 'picnic', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('Er lacht immer über meine Witze.', 'lachen', 'to laugh', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('Ich habe gestern einen spannenden Film gesehen.', 'Film', 'movie', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('Wir tanzen gern auf Partys.', 'tanzen', 'to dance', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('Sie trinkt morgens immer eine Tasse Kaffee.', 'Kaffee', 'coffee', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- +goose Down
DELETE FROM german_words
WHERE german_word IN ('Picknick', 'lachen', 'Film', 'tanzen', 'Kaffee');

