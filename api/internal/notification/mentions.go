package notification

import (
	"context"
	"regexp"
	"strings"
)

// Special mention constants
const (
	MentionChannel  = "@channel"
	MentionHere     = "@here"
	MentionEveryone = "@everyone"
)

// mrkdwnUserMention matches <@userId> format from the rich text editor
var mrkdwnUserMention = regexp.MustCompile(`<@([^>]+)>`)

// mrkdwnSpecialMention matches <!here>, <!channel>, <!everyone> from the rich text editor
var mrkdwnSpecialMention = regexp.MustCompile(`<!([^>]+)>`)

// mentionPattern matches @display_name patterns (plain text fallback)
// Matches @ followed by one or more words (display names can have spaces)
var mentionPattern = regexp.MustCompile(`@([A-Za-z][A-Za-z0-9 ]*[A-Za-z0-9]|[A-Za-z])`)

// UserResolver resolves display names to user IDs within a workspace
type UserResolver interface {
	ResolveDisplayNames(ctx context.Context, workspaceID string, names []string) (map[string]string, error)
}

// ParseMentions extracts and resolves mentions from message content.
// Supports both mrkdwn format (<@userId>, <!here>) and plain text (@DisplayName, @here).
// Returns a list of user IDs and special mention strings (@channel, @here, @everyone).
// Invalid mentions are silently ignored.
func ParseMentions(ctx context.Context, resolver UserResolver, workspaceID, content string) ([]string, error) {
	if content == "" {
		return nil, nil
	}

	var mentions []string
	seenUsers := make(map[string]bool)
	seenSpecial := make(map[string]bool)

	// First pass: extract mrkdwn-format user mentions <@userId>
	for _, match := range mrkdwnUserMention.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		userID := strings.TrimSpace(match[1])
		if userID != "" && !seenUsers[userID] {
			mentions = append(mentions, userID)
			seenUsers[userID] = true
		}
	}

	// Second pass: extract mrkdwn-format special mentions <!here>, <!channel>, <!everyone>
	for _, match := range mrkdwnSpecialMention.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(match[1]))
		switch name {
		case "channel":
			if !seenSpecial[MentionChannel] {
				mentions = append(mentions, MentionChannel)
				seenSpecial[MentionChannel] = true
			}
		case "here":
			if !seenSpecial[MentionHere] {
				mentions = append(mentions, MentionHere)
				seenSpecial[MentionHere] = true
			}
		case "everyone":
			if !seenSpecial[MentionEveryone] {
				mentions = append(mentions, MentionEveryone)
				seenSpecial[MentionEveryone] = true
			}
		}
	}

	// Third pass: plain text @DisplayName mentions (fallback for plain text content)
	var displayNames []string
	for _, match := range mentionPattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}

		name := strings.TrimSpace(match[1])
		lowerName := strings.ToLower(name)

		// Check for special mentions
		switch lowerName {
		case "channel":
			if !seenSpecial[MentionChannel] {
				mentions = append(mentions, MentionChannel)
				seenSpecial[MentionChannel] = true
			}
		case "here":
			if !seenSpecial[MentionHere] {
				mentions = append(mentions, MentionHere)
				seenSpecial[MentionHere] = true
			}
		case "everyone":
			if !seenSpecial[MentionEveryone] {
				mentions = append(mentions, MentionEveryone)
				seenSpecial[MentionEveryone] = true
			}
		default:
			// This could be a user display name
			displayNames = append(displayNames, name)
		}
	}

	// Resolve display names to user IDs
	if len(displayNames) > 0 && resolver != nil {
		resolved, err := resolver.ResolveDisplayNames(ctx, workspaceID, displayNames)
		if err != nil {
			// Don't fail on resolution errors, just skip unresolved mentions
			resolved = make(map[string]string)
		}

		for _, name := range displayNames {
			// Try exact match first, then case-insensitive
			if userID, ok := resolved[name]; ok && !seenUsers[userID] {
				mentions = append(mentions, userID)
				seenUsers[userID] = true
			} else if userID, ok := resolved[strings.ToLower(name)]; ok && !seenUsers[userID] {
				mentions = append(mentions, userID)
				seenUsers[userID] = true
			}
		}
	}

	return mentions, nil
}

// IsSpecialMention returns true if the mention is @channel, @here, or @everyone
func IsSpecialMention(mention string) bool {
	return mention == MentionChannel || mention == MentionHere || mention == MentionEveryone
}

// ResolveHereMentions replaces @here in mentions with the IDs of currently online
// channel members. Other mentions (including @channel and @everyone) pass through
// unchanged. The sender is excluded, and user IDs already in the mentions list
// are not duplicated.
func ResolveHereMentions(mentions []string, channelMemberIDs []string, senderID string, checker OnlineChecker, workspaceID string) []string {
	var result []string
	seen := make(map[string]bool)

	// First pass: collect already-mentioned user IDs
	for _, m := range mentions {
		if !IsSpecialMention(m) {
			seen[m] = true
		}
	}

	for _, m := range mentions {
		if m != MentionHere {
			result = append(result, m)
			continue
		}
		// Replace @here with online member IDs
		for _, memberID := range channelMemberIDs {
			if memberID == senderID {
				continue
			}
			if seen[memberID] {
				continue
			}
			if checker.IsUserOnline(workspaceID, memberID) {
				result = append(result, memberID)
				seen[memberID] = true
			}
		}
	}

	return result
}
