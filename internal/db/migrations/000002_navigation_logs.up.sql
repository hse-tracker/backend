CREATE TABLE navigation_logs ( -- Логи перемещения пользователей по вкладкам
     id SERIAL PRIMARY KEY,
     user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
     tab_name VARCHAR(255) NOT NULL,
     created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_nav_logs_user ON navigation_logs(user_id);
