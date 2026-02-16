-- Запросы написаны на основании нашей ОП

-- Создание факультета
INSERT INTO faculties (name)
VALUES ('Факультет компьютерных наук');

-- Создание ОП
INSERT INTO programs (faculty_id, name, max_course)
VALUES ((SELECT id FROM faculties WHERE name = 'Факультет компьютерных наук'),
        'Программная инженерия',
        4);

-- Создание предмета
INSERT INTO subjects (name, type, program_id)
VALUES ('Базы данных', 2, (SELECT id FROM programs WHERE name = 'Программная инженерия'));

-- Регистрация пользователя
INSERT INTO users (id,
                   first_name,
                   middle_name,
                   last_name,
                   program_id,
                   course_number,
                   language,
                   notifications_enabled)
VALUES (962788149, -- подтянется Telegram peer_id (user_id)
        'Денис',
        'Сергеевич',
        'Мельник',
        (SELECT id FROM programs WHERE name = 'Программная инженерия'),
        3,
        'ru', -- подтянется на основе языка в Telegram
        True);

-- Создание таблицы
INSERT INTO subject_tables (subject_id, url, created_by_user_id)
VALUES ((SELECT id FROM subjects WHERE name = 'Базы данных'),
        'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing', -- наша таблица
        962788149);

-- Подписка на таблицу
INSERT INTO subscriptions (user_id, subject_table_id)
VALUES (962788149,
        (SELECT id
         FROM subject_tables
         WHERE url =
               'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing'));

-- Отписка от таблицы
DELETE
FROM subscriptions
WHERE user_id = 962788149
  AND subject_table_id = (SELECT id
                          FROM subject_tables
                          WHERE url =
                                'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing');

-- Смена языка в настройках на английский
UPDATE users
SET language   = 'en',
    updated_at = NOW()
WHERE id = 962788149;

-- Переключение настройки уведомлений
UPDATE users
SET notifications_enabled = NOT notifications_enabled,
    updated_at            = NOW()
WHERE id = 962788149;

-- Создание оценки
INSERT INTO assessments (parent_id,
                         subscription_user_id,
                         subscription_subject_table_id,
                         value,
                         has_children)
VALUES (NULL,
        962788149,
        (SELECT id
         FROM subject_tables
         WHERE url =
               'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing'),
        '10',
        False);

-- Обновление существующей оценки
UPDATE assessments
SET value       = '4',
    assigned_at = NOW()
WHERE id = (SELECT id FROM assessments WHERE subscription_user_id = 962788149 LIMIT 1);

-- Обновление хеша таблицы при изменении
UPDATE subject_tables
SET last_hash      = 'HGLHFAGZNCHBROMGXTRXCRHRKCMAKRNC',
    last_parsed_at = NOW()
WHERE url = 'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing';

-- Транзакция: обновление хеша таблицы + вставка новых оценок (если первый шаг выполнился, а второй нет, то до следующего обновления таблицы мы не узнаем о новых оценках, что критично)
BEGIN;

    -- 1 - обновление хеша таблицы
UPDATE subject_tables
SET last_hash      = 'hash_update_transaction',
    last_parsed_at = NOW()
WHERE url = 'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing';

-- 2 - вставка новых оценок
INSERT INTO assessments (parent_id,
                         subscription_user_id,
                         subscription_subject_table_id,
                         value,
                         has_children)
VALUES (NULL,
        962788149,
        (SELECT id
         FROM subject_tables
         WHERE url =
               'https://docs.google.com/spreadsheets/d/1DJ8bEGosbj244zVPAsMS7SKKr3TG6sunQXjirl5tqi0/edit?usp=sharing'),
        '10',
        False);

COMMIT;
