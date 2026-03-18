-- User groups
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Users
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    full_name VARCHAR(255) NOT NULL,
    language VARCHAR(10) DEFAULT 'ru',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Subject tables
CREATE TABLE subjects (
    id BIGSERIAL PRIMARY KEY,
    creator_id BIGINT REFERENCES users(id) ON DELETE SET NULL, -- who created
    group_id BIGINT REFERENCES groups(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    group_connected BOOLEAN DEFAULT true,    -- checkbox "connect to the group"
    notifications BOOLEAN DEFAULT true,      -- checkbox "notifications"

    status VARCHAR(50) DEFAULT 'processing', -- 'processing', 'ready', 'error'
    error_message TEXT,                      -- only if (status == 'error')

    last_hash VARCHAR(255),                  -- last hash sum
    last_parsed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    name_column_index INTEGER,               -- column with student names
    data_start_row INTEGER                   -- row with 1st student
);

-- Grade structures
CREATE TABLE grade_structures (
    id BIGSERIAL PRIMARY KEY,
    subject_id BIGINT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES grade_structures(id) ON DELETE CASCADE, -- recursive FK to parent

    name VARCHAR(255) NOT NULL,    -- example: "Итог", "HW_AVG", "HW1"
    type VARCHAR(50) NOT NULL,     -- example: 'folder', 'value', 'formula', 'boolean'
    column_index INTEGER,          -- can be NULL if virtual

    weight DOUBLE PRECISION,       -- assessment weight, example: 0.5, 0.3, 0.1 
    display_formula VARCHAR(255)   -- readable formula
);

-- Student grades
CREATE TABLE student_grades (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grade_structure_id BIGINT NOT NULL REFERENCES grade_structures(id) ON DELETE CASCADE, -- link to assessment node

    value VARCHAR(255), -- example: "10", "5.5", "Н/Я", "Зачет"
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- student cant have 2 diff grades for same node
    UNIQUE (user_id, grade_structure_id)
);

-- Navigation logs
CREATE TABLE navigation_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tab_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);