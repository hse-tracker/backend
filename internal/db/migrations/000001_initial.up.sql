-- Факультеты
CREATE TABLE faculties
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

-- Образовательные программы
CREATE TABLE programs
(
    id         SERIAL PRIMARY KEY,
    faculty_id INT          NOT NULL REFERENCES faculties (id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    max_course INT          NOT NULL DEFAULT 4 CHECK (max_course IN (4, 6))
);

-- Пользователи
CREATE TABLE users
(
    id                    BIGINT PRIMARY KEY,
    -- замена full_name
    first_name            VARCHAR(255) NOT NULL,
    middle_name           VARCHAR(255), -- отчества может не быть
    last_name             VARCHAR(255) NOT NULL,
    program_id            INT          REFERENCES programs (id) ON DELETE SET NULL,
    course_number         INT          NOT NULL CHECK (course_number > 0 AND course_number <= 6),
    language              VARCHAR(10) DEFAULT 'ru',
    notifications_enabled BOOLEAN     DEFAULT TRUE,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_program ON users (program_id);

-- Дисциплины
CREATE TABLE subjects
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    type       SMALLINT     NOT NULL CHECK (type IN (1, 2)),
    program_id INT REFERENCES programs (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subjects_program ON subjects (program_id) WHERE program_id IS NOT NULL;
CREATE INDEX idx_subjects_name ON subjects (name);

-- Таблицы дисциплин
CREATE TABLE subject_tables
(
    id                 SERIAL PRIMARY KEY,
    subject_id         INT    NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,

    url                TEXT   NOT NULL UNIQUE,

    last_hash          VARCHAR(64),
    last_parsed_at     TIMESTAMPTZ,

    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);

-- Подписка пользователей на дисциплины
CREATE TABLE subscriptions
(
    user_id          BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    subject_table_id INT    NOT NULL REFERENCES subject_tables (id) ON DELETE CASCADE,

    created_at       TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (user_id, subject_table_id)
);

-- Оценки / формулы
CREATE TABLE assessments
(
    id                            SERIAL PRIMARY KEY,
    parent_id                     INT REFERENCES assessments (id) ON DELETE CASCADE,
    subscription_user_id          BIGINT,
    subscription_subject_table_id INT,

    has_children                  BOOLEAN,
    assigned_at                   TIMESTAMPTZ DEFAULT NOW(),
    value                         VARCHAR(255) NOT NULL,

    FOREIGN KEY (subscription_user_id, subscription_subject_table_id)
        REFERENCES subscriptions (user_id, subject_table_id)
        ON DELETE CASCADE
);
