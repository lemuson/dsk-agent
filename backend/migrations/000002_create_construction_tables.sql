
CREATE TYPE building_status AS ENUM ('design', 'construction', 'completed', 'suspended');
CREATE TYPE wall_material AS ENUM ('panel', 'monolith', 'brick', 'block');

CREATE TYPE finishing_type AS ENUM ('rough', 'white_box', 'turnkey');
CREATE TYPE apartment_status AS ENUM ('free', 'booked', 'sold');

CREATE TYPE progress_stage AS ENUM ('excavation', 'foundation', 'frame', 'roofing', 'finishing');
CREATE TYPE progress_status AS ENUM ('not_started', 'in_progress', 'completed', 'delayed');

CREATE TABLE IF NOT EXISTS residential_complexes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS buildings (
    id SERIAL PRIMARY KEY,
    residential_complex_id INT NOT NULL REFERENCES residential_complexes(id) ON DELETE CASCADE,
    address VARCHAR(255) NOT NULL,
    district VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    floors_count INT NOT NULL,
    planned_date DATE,
    actual_date DATE,
    status building_status NOT NULL,
    type_wall_material wall_material NOT NULL
);

CREATE TABLE IF NOT EXISTS apartments (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    number VARCHAR(50) NOT NULL,
    rooms INT NOT NULL,
    floor INT NOT NULL,
    area DECIMAL(10, 2) NOT NULL,
    price DECIMAL(15, 2) NOT NULL,
    type_finishing finishing_type NOT NULL,
    status apartment_status NOT NULL
);

CREATE TABLE IF NOT EXISTS construction_progress (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    stage_name progress_stage NOT NULL,
    planned_start_date DATE,
    actual_start_date DATE,
    planned_end_date DATE,
    actual_end_date DATE,
    status progress_status NOT NULL,
    completion_percentage INT DEFAULT 0,
    delay_reason VARCHAR(255),
    risk_level VARCHAR(20) CHECK (risk_level IS NULL OR risk_level IN ('low', 'medium', 'high')),
    delay_days INT CHECK (delay_days IS NULL OR delay_days >= 0)
);
