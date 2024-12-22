package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	nodeName "github.com/sing3demons/profile-service/constants"
	"github.com/sing3demons/profile-service/logger"
	"github.com/sing3demons/profile-service/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail    = errors.New("a user with that email already exists")
	ErrDuplicateUsername = errors.New("a user with that username already exists")
)

type User struct {
	ID           string   `json:"id"`
	Email        string   `json:"email,omitempty"`
	Username     string   `json:"username,omitempty"`
	Password     password `json:"password"`
	FirstName    string   `json:"first_name,omitempty"`
	LastName     string   `json:"last_name,omitempty"`
	DateOfBirth  string   `json:"date_of_birth,omitempty"`
	PhoneNumber  string   `json:"phone_number,omitempty"`
	Gender       string   `json:"gender,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	CreatedBy    string   `json:"created_by,omitempty"`
	UpdatedBy    string   `json:"updated_by,omitempty"`
	DisplayName  string   `json:"display_name,omitempty"`
	ProfileImage string   `json:"profile_image,omitempty"`
}

type password struct {
	text *string
	hash []byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.text = &text
	p.hash = hash

	return nil
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

type UserStore struct {
	db *sql.DB
}

type ProcessLog struct {
	Query  string      `json:"Query,omitempty"`
	Params interface{} `json:"Params,omitempty"`
	Return interface{} `json:"Return,omitempty"`
}

func (s *UserStore) Create(ctx context.Context, tx *sql.Tx, user *User, detailLog logger.DetailLog, summaryLog logger.SummaryLog) error {
	query := "INSERT INTO Profile (username, password, email) VALUES ($1, $2, $3) RETURNING id, created_at"

	cmd := "create_user"
	invoke := utils.GenerateXTid(cmd)

	processLog := ProcessLog{
		Query:  query,
		Params: user,
	}

	detailLog.AddOutputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Query), processLog)
	detailLog.End()

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	row := tx.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Password.hash,
		user.Email,
	)

	err := row.Scan(
		&user.ID,
		&user.CreatedAt,
	)

	if err != nil {
		detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, nil, err)
		summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", err.Error())
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}

	detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, user, user)
	summaryLog.AddSuccessBlock(nodeName.POSTGRES, cmd, "200", "success")

	return nil
}

func (s *UserStore) GetByID(ctx context.Context, userID string, detailLog logger.DetailLog, summaryLog logger.SummaryLog) (*User, error) {
	query := `SELECT id, email, username, created_at, updated_at, first_name, last_name, date_of_birth, phone_number FROM Profile WHERE id = $1`

	cmd := "get_user_by_id"
	invoke := utils.GenerateXTid(cmd)

	processLog := ProcessLog{
		Query: strings.Replace(query, "$1", fmt.Sprintf("'%s'", userID), 1),
	}

	detailLog.AddOutputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Query), processLog)
	detailLog.End()

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &User{}
	err := s.db.QueryRowContext(
		ctx,
		query,
		userID,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.PhoneNumber,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, err.Error(), err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", err.Error())
			return nil, ErrNotFound
		case sql.ErrConnDone:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, err.Error(), err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", sql.ErrConnDone.Error())
			return nil, err
		case sql.ErrTxDone:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, err.Error(), err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", sql.ErrTxDone.Error())
			return nil, err
		}
	}

	processLog = ProcessLog{
		Return: user,
	}

	detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Return), processLog)
	summaryLog.AddSuccessBlock(nodeName.POSTGRES, cmd, "200", "success")

	return user, nil
}

func (s *UserStore) CreateAndInvite(ctx context.Context, user *User, token string, invitationExp time.Duration, detailLog logger.DetailLog, summaryLog logger.SummaryLog) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		if err := s.Create(ctx, tx, user, detailLog, summaryLog); err != nil {
			return err
		}

		if err := s.createUserInvitation(ctx, tx, token, invitationExp, user.ID); err != nil {
			return err
		}

		return nil
	})
}

func (s *UserStore) getUserFromInvitation(ctx context.Context, tx *sql.Tx, token string) (*User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.created_at
		FROM users u
		JOIN user_invitations ui ON u.id = ui.user_id
		WHERE ui.token = $1 AND ui.expiry > $2
	`

	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &User{}
	err := tx.QueryRowContext(ctx, query, hashToken, time.Now()).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (s *UserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Duration, userID string) error {
	query := `INSERT INTO user_invitations (token, user_id, expiry) VALUES ($1, $2, $3)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, token, userID, time.Now().Add(exp))
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) update(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `UPDATE users SET username = $1, email = $2 WHERE id = $3`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, user.Username, user.Email, user.ID)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) deleteUserInvitations(ctx context.Context, tx *sql.Tx, userID string) error {
	query := `DELETE FROM user_invitations WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) Delete(ctx context.Context, userID string, detailLog logger.DetailLog, summaryLog logger.SummaryLog) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		if err := s.delete(ctx, tx, userID, detailLog, summaryLog); err != nil {
			return err
		}

		if err := s.deleteUserInvitations(ctx, tx, userID); err != nil {
			return err
		}

		return nil
	})
}

func (s *UserStore) delete(ctx context.Context, tx *sql.Tx, id string, detailLog logger.DetailLog, summaryLog logger.SummaryLog) error {
	query := `DELETE FROM Profile WHERE id = $1`
	cmd := "delete_user"
	invoke := utils.GenerateXTid(cmd)

	processLog := ProcessLog{
		Query: strings.Replace(query, "$1", fmt.Sprintf("'%s'", id), 1),
	}
	detailLog.AddOutputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Query), processLog)
	detailLog.End()

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, nil, err)
		summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", err.Error())
		return err
	}

	processLog = ProcessLog{
		Return: result,
	}

	detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Return), processLog)
	summaryLog.AddSuccessBlock(nodeName.POSTGRES, cmd, "200", "success")

	return nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string, detailLog logger.DetailLog, summaryLog logger.SummaryLog) (*User, error) {
	query := `SELECT id, username, email, password, created_at FROM Profile WHERE email = $1`

	cmd := "select_user_by_email"
	invoke := utils.GenerateXTid(cmd)

	processLog := ProcessLog{
		Query: strings.Replace(query, "$1", fmt.Sprintf("'%s'", email), 1),
	}
	detailLog.AddOutputRequest(nodeName.POSTGRES, cmd, invoke, fmt.Sprintf("%v", processLog.Query), processLog)
	detailLog.End()

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.CreatedAt,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, err.Error(), err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", sql.ErrNoRows.Error())
			return nil, ErrNotFound
		case sql.ErrConnDone:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, nil, err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", sql.ErrConnDone.Error())
			return nil, err
		case sql.ErrTxDone:
			detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, nil, err)
			summaryLog.AddErrorBlock(nodeName.POSTGRES, cmd, "500", sql.ErrTxDone.Error())
			return nil, err
		}
	}
	processLog = ProcessLog{
		Return: user,
	}
	detailLog.AddInputRequest(nodeName.POSTGRES, cmd, invoke, "", processLog)
	summaryLog.AddSuccessBlock(nodeName.POSTGRES, cmd, "200", "success")

	return user, nil
}
