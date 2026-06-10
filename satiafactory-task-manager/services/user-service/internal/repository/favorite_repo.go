package repository

import (
	"database/sql"
	"fmt"

	"github.com/dratbo/satisfactory-task-manager/user-service/internal/models"
)

type FavoriteRepository struct {
	db *sql.DB
}

func NewFavoriteRepository(db *sql.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) List(userID int64) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.email, u.created_at
		FROM favorite_users f
		JOIN users u ON u.id = f.favorite_user_id
		WHERE f.user_id = $1
		ORDER BY u.username`, userID)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
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

func (r *FavoriteRepository) Add(userID, favoriteID int64) error {
	_, err := r.db.Exec(
		`INSERT INTO favorite_users (user_id, favorite_user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, favoriteID,
	)
	return err
}

func (r *FavoriteRepository) Remove(userID, favoriteID int64) error {
	_, err := r.db.Exec(
		`DELETE FROM favorite_users WHERE user_id = $1 AND favorite_user_id = $2`,
		userID, favoriteID,
	)
	return err
}

func (r *FavoriteRepository) IsFavorite(userID, favoriteID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM favorite_users WHERE user_id = $1 AND favorite_user_id = $2)`,
		userID, favoriteID,
	).Scan(&exists)
	return exists, err
}
