CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    balance NUMERIC(10, 2) NOT NULL DEFAULT 0.00
);

INSERT INTO users (name, email, balance) VALUES
('Sender User', 'sender@example.com', 1000.00)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (name, email, balance) VALUES
('Receiver User', 'receiver@example.com', 500.00)
ON CONFLICT (email) DO NOTHING;
