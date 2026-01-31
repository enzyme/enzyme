package workspace

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/feather/api/internal/auth"
	"github.com/oklog/ulid/v2"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrSlugAlreadyInUse  = errors.New("slug already in use")
	ErrMembershipExists  = errors.New("user is already a member")
	ErrNotAMember        = errors.New("user is not a member of this workspace")
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrInviteMaxUsed     = errors.New("invite has reached max uses")
	ErrCannotRemoveOwner = errors.New("cannot remove workspace owner")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, workspace *Workspace, ownerUserID string) error {
	workspace.ID = ulid.Make().String()
	now := time.Now().UTC()
	workspace.CreatedAt = now
	workspace.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspaces (id, slug, name, icon_url, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, workspace.ID, workspace.Slug, workspace.Name, workspace.IconURL, workspace.Settings, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrSlugAlreadyInUse
		}
		return err
	}

	// Add owner membership
	membershipID := ulid.Make().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_memberships (id, user_id, workspace_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, membershipID, ownerUserID, workspace.ID, RoleOwner, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Workspace, error) {
	return r.scanWorkspace(r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, icon_url, settings, created_at, updated_at
		FROM workspaces WHERE id = ?
	`, id))
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	return r.scanWorkspace(r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, icon_url, settings, created_at, updated_at
		FROM workspaces WHERE slug = ?
	`, slug))
}

func (r *Repository) Update(ctx context.Context, workspace *Workspace) error {
	workspace.UpdatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE workspaces SET slug = ?, name = ?, icon_url = ?, settings = ?, updated_at = ?
		WHERE id = ?
	`, workspace.Slug, workspace.Name, workspace.IconURL, workspace.Settings, workspace.UpdatedAt.Format(time.RFC3339), workspace.ID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrSlugAlreadyInUse
		}
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrWorkspaceNotFound
	}
	return nil
}

func (r *Repository) GetMembership(ctx context.Context, userID, workspaceID string) (*Membership, error) {
	var m Membership
	var displayNameOverride sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, workspace_id, role, display_name_override, created_at, updated_at
		FROM workspace_memberships WHERE user_id = ? AND workspace_id = ?
	`, userID, workspaceID).Scan(&m.ID, &m.UserID, &m.WorkspaceID, &m.Role, &displayNameOverride, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotAMember
	}
	if err != nil {
		return nil, err
	}

	if displayNameOverride.Valid {
		m.DisplayNameOverride = &displayNameOverride.String
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &m, nil
}

func (r *Repository) AddMember(ctx context.Context, userID, workspaceID, role string) (*Membership, error) {
	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workspace_memberships (id, user_id, workspace_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, userID, workspaceID, role, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrMembershipExists
		}
		return nil, err
	}

	return &Membership{
		ID:          id,
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        role,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *Repository) RemoveMember(ctx context.Context, userID, workspaceID string) error {
	// Check if owner
	var role string
	err := r.db.QueryRowContext(ctx, `
		SELECT role FROM workspace_memberships WHERE user_id = ? AND workspace_id = ?
	`, userID, workspaceID).Scan(&role)
	if err == sql.ErrNoRows {
		return ErrNotAMember
	}
	if err != nil {
		return err
	}
	if role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	_, err = r.db.ExecContext(ctx, `
		DELETE FROM workspace_memberships WHERE user_id = ? AND workspace_id = ?
	`, userID, workspaceID)
	return err
}

func (r *Repository) UpdateMemberRole(ctx context.Context, userID, workspaceID, newRole string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE workspace_memberships SET role = ?, updated_at = ?
		WHERE user_id = ? AND workspace_id = ?
	`, newRole, now.Format(time.RFC3339), userID, workspaceID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotAMember
	}
	return nil
}

func (r *Repository) ListMembers(ctx context.Context, workspaceID string) ([]MemberWithUser, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT wm.id, wm.user_id, wm.workspace_id, wm.role, wm.display_name_override, wm.created_at, wm.updated_at,
		       u.email, u.display_name, u.avatar_url
		FROM workspace_memberships wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = ?
		ORDER BY wm.created_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []MemberWithUser
	for rows.Next() {
		var m MemberWithUser
		var displayNameOverride, avatarURL sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(&m.ID, &m.UserID, &m.WorkspaceID, &m.Role, &displayNameOverride, &createdAt, &updatedAt,
			&m.Email, &m.DisplayName, &avatarURL)
		if err != nil {
			return nil, err
		}

		if displayNameOverride.Valid {
			m.DisplayNameOverride = &displayNameOverride.String
		}
		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		members = append(members, m)
	}

	return members, rows.Err()
}

func (r *Repository) GetWorkspacesForUser(req *http.Request, userID string) ([]auth.WorkspaceSummary, error) {
	rows, err := r.db.QueryContext(req.Context(), `
		SELECT w.id, w.slug, w.name, w.icon_url, wm.role
		FROM workspaces w
		JOIN workspace_memberships wm ON wm.workspace_id = w.id
		WHERE wm.user_id = ?
		ORDER BY w.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []auth.WorkspaceSummary
	for rows.Next() {
		var ws auth.WorkspaceSummary
		var iconURL sql.NullString

		err := rows.Scan(&ws.ID, &ws.Slug, &ws.Name, &iconURL, &ws.Role)
		if err != nil {
			return nil, err
		}

		if iconURL.Valid {
			ws.IconURL = &iconURL.String
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, rows.Err()
}

// Invite methods
func (r *Repository) CreateInvite(ctx context.Context, invite *Invite) error {
	invite.ID = ulid.Make().String()
	if invite.Code == "" {
		invite.Code = generateInviteCode()
	}
	now := time.Now().UTC()
	invite.CreatedAt = now

	var expiresAt *string
	if invite.ExpiresAt != nil {
		s := invite.ExpiresAt.Format(time.RFC3339)
		expiresAt = &s
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workspace_invites (id, workspace_id, code, invited_email, role, created_by, max_uses, use_count, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, invite.ID, invite.WorkspaceID, invite.Code, invite.InvitedEmail, invite.Role, invite.CreatedBy, invite.MaxUses, 0, expiresAt, now.Format(time.RFC3339))
	return err
}

func (r *Repository) GetInviteByCode(ctx context.Context, code string) (*Invite, error) {
	var invite Invite
	var invitedEmail, createdBy, expiresAt sql.NullString
	var maxUses sql.NullInt64
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, code, invited_email, role, created_by, max_uses, use_count, expires_at, created_at
		FROM workspace_invites WHERE code = ?
	`, code).Scan(&invite.ID, &invite.WorkspaceID, &invite.Code, &invitedEmail, &invite.Role, &createdBy, &maxUses, &invite.UseCount, &expiresAt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, err
	}

	if invitedEmail.Valid {
		invite.InvitedEmail = &invitedEmail.String
	}
	if createdBy.Valid {
		invite.CreatedBy = &createdBy.String
	}
	if maxUses.Valid {
		v := int(maxUses.Int64)
		invite.MaxUses = &v
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		invite.ExpiresAt = &t
	}
	invite.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &invite, nil
}

func (r *Repository) IncrementInviteUseCount(ctx context.Context, inviteID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE workspace_invites SET use_count = use_count + 1 WHERE id = ?
	`, inviteID)
	return err
}

func (r *Repository) AcceptInvite(ctx context.Context, code string, userID string) (*Workspace, error) {
	invite, err := r.GetInviteByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Check expiry
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	// Check max uses
	if invite.MaxUses != nil && invite.UseCount >= *invite.MaxUses {
		return nil, ErrInviteMaxUsed
	}

	// Add member
	_, err = r.AddMember(ctx, userID, invite.WorkspaceID, invite.Role)
	if err != nil && !errors.Is(err, ErrMembershipExists) {
		return nil, err
	}

	// Increment use count
	if err := r.IncrementInviteUseCount(ctx, invite.ID); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, invite.WorkspaceID)
}

func (r *Repository) scanWorkspace(row *sql.Row) (*Workspace, error) {
	var w Workspace
	var iconURL sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&w.ID, &w.Slug, &w.Name, &iconURL, &w.Settings, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, err
	}

	if iconURL.Valid {
		w.IconURL = &iconURL.String
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	w.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &w, nil
}

func generateInviteCode() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
