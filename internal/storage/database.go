package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	dbase "github.com/hse-tracker/backend/internal/db"
	"github.com/huandu/go-sqlbuilder"
	_ "github.com/lib/pq"
)

type User struct {
	ID                   int64     `db:"id"`
	FirstName            string    `db:"first_name"`
	MiddleName           string    `db:"middle_name"`
	LastName             string    `db:"last_name"`
	ProgramID            *int      `db:"program_id"`
	CourseNumber         int       `db:"course_number"`
	Language             string    `db:"language"`
	NotificationsEnabled bool      `db:"notifications_enabled"`
	CreatedAt            time.Time `db:"created_at"`
}

type Faculty struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type Program struct {
	ID        int    `db:"id"`
	FacultyID int    `db:"faculty_id"`
	Name      string `db:"name"`
	MaxCourse int    `db:"max_course"`
}

type Subject struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	Type      int    `db:"type"`
	ProgramID *int   `db:"program_id"`
}

type SubjectTable struct {
	ID           int       `db:"id"`
	SubjectID    int       `db:"subject_id"`
	URL          string    `db:"url"`
	LastHash     string    `db:"last_hash"`
	LastParsedAt time.Time `db:"last_parsed_at"`
}

type Storage struct {
	db *sql.DB
}

func New(dsn string) (*Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		log.Printf("Waiting for database... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("database unreachable after retries: %w", pingErr)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("could not create postgres driver: %w", err)
	}

	sourceInstance, err := iofs.New(dbase.MigrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs source for migrations: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"file://migrations",
		sourceInstance,
		"postgres",
		driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	return &Storage{db: db}, nil
}

func (s *Storage) Close() {
	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			return
		}
	}
}

// UpsertUser Вставка пользователя
func (s *Storage) UpsertUser(u User) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("users").
		Cols("id", "first_name", "middle_name", "last_name", "program_id", "course_number", "language", "notifications_enabled").
		Values(u.ID, u.FirstName, u.MiddleName, u.LastName, u.ProgramID, u.CourseNumber, u.Language, u.NotificationsEnabled)

	ib.SQL(`
		ON CONFLICT (id) DO UPDATE SET
		first_name = excluded.first_name,
		middle_name = excluded.middle_name,
		last_name = excluded.last_name,
		program_id = excluded.program_id,
		course_number = excluded.course_number,
		updated_at = NOW()
	`)

	query, args := ib.Build()
	_, err := s.db.Exec(query, args...)
	return err
}

// GetUser Получение пользователя
func (s *Storage) GetUser(telegramID int64) (*User, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "first_name", "middle_name", "last_name", "program_id", "course_number", "language", "notifications_enabled").
		From("users").
		Where(sb.Equal("id", telegramID))

	query, args := sb.Build()
	row := s.db.QueryRow(query, args...)

	var u User
	var middleName sql.NullString

	if err := row.Scan(&u.ID, &u.FirstName, &middleName, &u.LastName, &u.ProgramID, &u.CourseNumber, &u.Language, &u.NotificationsEnabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	u.MiddleName = middleName.String

	return &u, nil
}

// GetFaculties Получение факультетов
func (s *Storage) GetFaculties() ([]Faculty, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "name").From("faculties").OrderByAsc("name")

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var faculties []Faculty
	for rows.Next() {
		var f Faculty
		if err := rows.Scan(&f.ID, &f.Name); err != nil {
			return nil, err
		}
		faculties = append(faculties, f)
	}
	return faculties, nil
}

// GetPrograms Получение образовательных программ
func (s *Storage) GetPrograms(facultyID int) ([]Program, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "faculty_id", "name", "max_course").
		From("programs").
		Where(sb.Equal("faculty_id", facultyID)).
		OrderByAsc("name")

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var programs []Program
	for rows.Next() {
		var p Program
		if err := rows.Scan(&p.ID, &p.FacultyID, &p.Name, &p.MaxCourse); err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

// GetAvailableSubjects Получение доступных дисциплин
func (s *Storage) GetAvailableSubjects(programID int) ([]Subject, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "name", "type", "program_id").
		From("subjects").
		Where(
			sb.Or(
				sb.Equal("type", 2),
				sb.And(sb.Equal("type", 1), sb.Equal("program_id", programID)),
			),
		).
		OrderByAsc("name")

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var subjects []Subject
	for rows.Next() {
		var sub Subject
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.Type, &sub.ProgramID); err != nil {
			return nil, err
		}
		subjects = append(subjects, sub)
	}
	return subjects, nil
}

// CreateSubject Создать дисциплину
func (s *Storage) CreateSubject(name string, subType int, programID *int) (int, error) {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("subjects").
		Cols("name", "type", "program_id").
		Values(name, subType, programID)

	query, args := ib.Build()
	query += " RETURNING id"

	var id int
	err := s.db.QueryRow(query, args...).Scan(&id)
	return id, err
}

// FindOrCreateTable Найти или создать таблицу дисциплины
func (s *Storage) FindOrCreateTable(subjectID int, url string, createdBy int64) (int, error) {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("subject_tables").
		Cols("subject_id", "url", "created_by_user_id").
		Values(subjectID, url, createdBy)

	ib.SQL("ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url RETURNING id")

	query, args := ib.Build()

	var tableID int
	if err := s.db.QueryRow(query, args...).Scan(&tableID); err != nil {
		return 0, fmt.Errorf("find or create table error: %w", err)
	}
	return tableID, nil
}

// SubscribeUser Подписать пользователя на таблицу дисциплины
func (s *Storage) SubscribeUser(userID int64, tableID int) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("subscriptions").
		Cols("user_id", "subject_table_id").
		Values(userID, tableID)
	ib.SQL("ON CONFLICT (user_id, subject_table_id) DO NOTHING")

	query, args := ib.Build()
	_, err := s.db.Exec(query, args...)
	return err
}

// GetUserSubscriptions Получить подписки пользователя
func (s *Storage) GetUserSubscriptions(userID int64) ([]SubjectTable, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("st.id", "st.subject_id", "st.url", "st.last_parsed_at").
		From("subscriptions s").
		Join("subject_tables st", "s.subject_table_id = st.id").
		Where(sb.Equal("s.user_id", userID))

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var tables []SubjectTable
	for rows.Next() {
		var t SubjectTable
		var lastParsed sql.NullTime
		if err := rows.Scan(&t.ID, &t.SubjectID, &t.URL, &lastParsed); err != nil {
			return nil, err
		}
		t.LastParsedAt = lastParsed.Time
		tables = append(tables, t)
	}
	return tables, nil
}

func (s *Storage) GetAllActiveTables() ([]SubjectTable, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "subject_id", "url", "last_hash").
		From("subject_tables")

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var tables []SubjectTable
	for rows.Next() {
		var t SubjectTable
		var hash sql.NullString
		if err := rows.Scan(&t.ID, &t.SubjectID, &t.URL, &hash); err != nil {
			return nil, err
		}
		t.LastHash = hash.String
		tables = append(tables, t)
	}
	return tables, nil
}

func (s *Storage) UpdateTableHash(tableID int, newHash string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("subject_tables").
		Set(
			ub.Assign("last_hash", newHash),
			ub.Assign("last_parsed_at", "NOW()"),
		).
		Where(ub.Equal("id", tableID))

	query, args := ub.Build()
	_, err := s.db.Exec(query, args...)
	return err
}

func (s *Storage) GetSubscribersForTable(tableID int) ([]int64, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("user_id").From("subscriptions").Where(sb.Equal("subject_table_id", tableID))

	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var userIDs []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, nil
}

type NavigationLog struct {
	UserID    int64     `db:"user_id"`
	TabName   string    `db:"tab_name"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Storage) LogNavigation(userID int64, tabName string) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("navigation_logs").
		Cols("user_id", "tab_name").
		Values(userID, tabName)
	query, args := ib.Build()
	_, err := s.db.Exec(query, args...)
	return err
}

func (s *Storage) GetAllNavigationLogs() ([]NavigationLog, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("user_id", "tab_name", "created_at").From("navigation_logs").OrderByDesc("created_at")
	query, args := sb.Build()
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var logs []NavigationLog
	for rows.Next() {
		var l NavigationLog
		if err := rows.Scan(&l.UserID, &l.TabName, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}
