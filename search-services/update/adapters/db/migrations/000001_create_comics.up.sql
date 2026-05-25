CREATE TABLE comics (
    id SERIAL PRIMARY KEY,
    comics_id INT,
    image_url TEXT,
    key_words TEXT[]
);