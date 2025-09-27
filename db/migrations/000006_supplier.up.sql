CREATE TABLE suppliers(
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    phone VARCHAR(50),
    email VARCHAR(50),
    address VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    business_id INTEGER NOT NULL REFERENCES business(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
