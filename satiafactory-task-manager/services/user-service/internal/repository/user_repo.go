package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/dratbo/satisfactory-task-manager/user-service/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("username or email already exists")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.Password = string(hashed)

	query := `INSERT INTO users (username, email, password, created_at) VALUES ($1, $2, $3, NOW()) RETURNING id, created_at`
	err = r.db.QueryRow(query, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		// Check for unique violation (PostgreSQL code 23505)
		if err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"` ||
			err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, email, password, created_at FROM users WHERE username = $1`
	err := r.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Search(query string) ([]models.User, error) {
	if query == "" {
		return r.ListAll()
	}
	rows, err := r.db.Query(
		`SELECT id, username, email, created_at FROM users WHERE username ILIKE '%' || $1 || '%' ORDER BY username LIMIT 20`,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) ListAll() ([]models.User, error) {
	rows, err := r.db.Query(`SELECT id, username, email, created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) FindByID(id int64) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, email, created_at FROM users WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func CheckPassword(hashed, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}
