-- The business's address, city and phone (P12), printed on the receipt and
-- the cash closing report. Empty means not registered: the domain trims them
-- and checks their characters, the database only their length. No data
-- moves: every existing tenant starts without them.

-- +goose Up
ALTER TABLE tenants
    ADD COLUMN address text NOT NULL DEFAULT '' CHECK (char_length(address) <= 200),
    ADD COLUMN city    text NOT NULL DEFAULT '' CHECK (char_length(city) <= 100),
    ADD COLUMN phone   text NOT NULL DEFAULT '' CHECK (char_length(phone) <= 30);

-- +goose Down
ALTER TABLE tenants
    DROP COLUMN phone,
    DROP COLUMN city,
    DROP COLUMN address;
