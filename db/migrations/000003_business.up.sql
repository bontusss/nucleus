

CREATE TYPE payment_type AS ENUM ('cash', 'pos', 'room_charge', 'transfer');

-- Create the business table
CREATE TABLE businesses (
    id SERIAL PRIMARY KEY,
    owner_id INTEGER NOT NULL REFERENCES admins(id),
    name VARCHAR(50) NOT NULL UNIQUE,
    motto VARCHAR(50),
    email VARCHAR(50) UNIQUE,
    website VARCHAR(20),
    tax_id VARCHAR(100),
    vat_number VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    logo_url VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_business_name ON businesses (name);
CREATE INDEX idx_business_email ON businesses (email);
CREATE INDEX idx_business_tax_id ON businesses (tax_id);
CREATE INDEX idx_business_owner_id ON businesses (owner_id);


CREATE TABLE branches (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL,
    name VARCHAR(50) NOT NULL,
    address VARCHAR(20),
    phone VARCHAR(15),
    email VARCHAR(20),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (business_id) REFERENCES businesses(id) ON DELETE CASCADE
);

CREATE TABLE taxes (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    name VARCHAR(50) NOT NULL,
    rate NUMERIC(4,2) NOT NULL,
    code VARCHAR(10) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    note TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_branch_business_id ON branches (business_id);
CREATE INDEX idx_branch_name ON branches (name);
