package moderation

// FilterOptions carries context for ban-hide and block filtering in message queries.
// When non-nil, messages from banned users (with hide_messages=1) and blocked users
// are excluded from results. Reactions and thread participants from those users are
// also filtered.
type FilterOptions struct {
	WorkspaceID      string // Required for ban-hide (workspace_bans) and block (user_blocks) filters
	RequestingUserID string // Required for block filter (blocker_id)
}

// FilterSQL returns SQL WHERE clause fragments and args for ban-hide and block filtering.
// userCol is the column reference for the user to filter (e.g., "m.user_id", "user_id").
// Returns empty string and nil args when filter is nil or has no workspace context.
func FilterSQL(filter *FilterOptions, userCol string) (string, []interface{}) {
	if filter == nil || filter.WorkspaceID == "" {
		return "", nil
	}
	var sql string
	var args []interface{}

	// Ban-hide filter: exclude messages from banned users with hide_messages=1
	sql += ` AND ` + userCol + ` NOT IN (
		SELECT wb.user_id FROM workspace_bans wb
		WHERE wb.workspace_id = ? AND wb.hide_messages = 1
		AND (wb.expires_at IS NULL OR wb.expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`
	args = append(args, filter.WorkspaceID)

	// Block filter: exclude messages from users the requester has blocked
	if filter.RequestingUserID != "" {
		sql += ` AND ` + userCol + ` NOT IN (
			SELECT ub.blocked_id FROM user_blocks ub
			WHERE ub.workspace_id = ? AND ub.blocker_id = ?
		)`
		args = append(args, filter.WorkspaceID, filter.RequestingUserID)
	}

	return sql, args
}
