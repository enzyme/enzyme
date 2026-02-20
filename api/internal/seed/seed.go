package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/enzyme/api/internal/auth"
	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/user"
	"github.com/enzyme/api/internal/workspace"
)

// Run populates the database with seed data for development.
// It is idempotent — if data already exists, it logs and returns nil.
func Run(ctx context.Context, db *sql.DB) error {
	// Idempotency check
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE email = 'alice@example.com'`).Scan(&count); err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if count > 0 {
		slog.Info("database already seeded, skipping")
		return nil
	}

	slog.Info("seeding database...")

	userRepo := user.NewRepository(db)
	workspaceRepo := workspace.NewRepository(db)
	channelRepo := channel.NewRepository(db)
	messageRepo := message.NewRepository(db)

	// Hash password once (bcrypt cost 4 for speed)
	hash, err := auth.HashPassword("password", 4)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// --- Users ---
	type seedUser struct {
		email       string
		displayName string
	}
	seedUsers := []seedUser{
		{"alice@example.com", "Alice Chen"},
		{"bob@example.com", "Bob Martinez"},
		{"carol@example.com", "Carol Williams"},
		{"dave@example.com", "Dave Johnson"},
		{"eve@example.com", "Eve Kim"},
		{"frank@example.com", "Frank O'Brien"},
		{"grace@example.com", "Grace Patel"},
		{"hank@example.com", "Hank Nguyen"},
	}

	users := make([]*user.User, len(seedUsers))
	for i, su := range seedUsers {
		u, err := userRepo.Create(ctx, user.CreateUserInput{
			Email:        su.email,
			DisplayName:  su.displayName,
			PasswordHash: hash,
		})
		if err != nil {
			return fmt.Errorf("create user %s: %w", su.email, err)
		}
		users[i] = u
		slog.Info("created user", "email", su.email)
	}

	alice, bob, carol, dave, eve, frank, grace, hank := users[0], users[1], users[2], users[3], users[4], users[5], users[6], users[7]

	// --- Workspace 1: Acme Corp ---
	ws1 := &workspace.Workspace{
		Name:     "Acme Corp",
		Settings: workspace.DefaultSettings().ToJSON(),
	}
	if err := workspaceRepo.Create(ctx, ws1, alice.ID); err != nil {
		return fmt.Errorf("create workspace Acme Corp: %w", err)
	}
	slog.Info("created workspace", "name", ws1.Name)

	// Add members to Acme Corp
	ws1Members := []struct {
		userID string
		role   string
	}{
		{bob.ID, workspace.RoleAdmin},
		{carol.ID, workspace.RoleMember},
		{dave.ID, workspace.RoleMember},
		{eve.ID, workspace.RoleMember},
		{frank.ID, workspace.RoleMember},
		{grace.ID, workspace.RoleGuest},
		{hank.ID, workspace.RoleMember},
	}
	for _, m := range ws1Members {
		if _, err := workspaceRepo.AddMember(ctx, m.userID, ws1.ID, m.role); err != nil {
			return fmt.Errorf("add member to Acme Corp: %w", err)
		}
	}

	// --- Workspace 2: Side Project ---
	ws2 := &workspace.Workspace{
		Name:     "Side Project",
		Settings: workspace.DefaultSettings().ToJSON(),
	}
	if err := workspaceRepo.Create(ctx, ws2, hank.ID); err != nil {
		return fmt.Errorf("create workspace Side Project: %w", err)
	}
	slog.Info("created workspace", "name", ws2.Name)

	// Add members to Side Project
	for _, m := range []struct {
		userID string
		role   string
	}{
		{alice.ID, workspace.RoleAdmin},
		{bob.ID, workspace.RoleMember},
		{carol.ID, workspace.RoleMember},
	} {
		if _, err := workspaceRepo.AddMember(ctx, m.userID, ws2.ID, m.role); err != nil {
			return fmt.Errorf("add member to Side Project: %w", err)
		}
	}

	// --- Channels in Acme Corp ---
	posterRole := channel.ChannelRolePoster

	// #general (default channel)
	acmeGeneral, err := channelRepo.CreateDefaultChannel(ctx, ws1.ID, alice.ID)
	if err != nil {
		return fmt.Errorf("create #general in Acme Corp: %w", err)
	}
	// Add all members to #general (alice is already added as creator)
	acmeAllExceptAlice := []*user.User{bob, carol, dave, eve, frank, grace, hank}
	for _, u := range acmeAllExceptAlice {
		if _, err := channelRepo.AddMember(ctx, u.ID, acmeGeneral.ID, &posterRole); err != nil {
			return fmt.Errorf("add %s to #general: %w", u.DisplayName, err)
		}
	}

	// #random (public)
	acmeRandom := &channel.Channel{
		WorkspaceID: ws1.ID,
		Name:        "random",
		Type:        channel.TypePublic,
	}
	if err := channelRepo.Create(ctx, acmeRandom, alice.ID); err != nil {
		return fmt.Errorf("create #random: %w", err)
	}
	for _, u := range acmeAllExceptAlice {
		if _, err := channelRepo.AddMember(ctx, u.ID, acmeRandom.ID, &posterRole); err != nil {
			return fmt.Errorf("add %s to #random: %w", u.DisplayName, err)
		}
	}

	// #admins-only (private)
	adminsOnly := &channel.Channel{
		WorkspaceID: ws1.ID,
		Name:        "admins-only",
		Type:        channel.TypePrivate,
	}
	desc := "Private channel for workspace admins"
	adminsOnly.Description = &desc
	if err := channelRepo.Create(ctx, adminsOnly, alice.ID); err != nil {
		return fmt.Errorf("create #admins-only: %w", err)
	}
	if _, err := channelRepo.AddMember(ctx, bob.ID, adminsOnly.ID, &posterRole); err != nil {
		return fmt.Errorf("add Bob to #admins-only: %w", err)
	}

	// DMs in Acme Corp
	dmAliceBob, err := channelRepo.CreateDM(ctx, ws1.ID, []string{alice.ID, bob.ID})
	if err != nil {
		return fmt.Errorf("create DM Alice-Bob: %w", err)
	}
	dmCarolDave, err := channelRepo.CreateDM(ctx, ws1.ID, []string{carol.ID, dave.ID})
	if err != nil {
		return fmt.Errorf("create DM Carol-Dave: %w", err)
	}
	gdmAliceCarolEve, err := channelRepo.CreateDM(ctx, ws1.ID, []string{alice.ID, carol.ID, eve.ID})
	if err != nil {
		return fmt.Errorf("create group DM Alice-Carol-Eve: %w", err)
	}

	slog.Info("created Acme Corp channels")

	// --- Channels in Side Project ---
	spGeneral, err := channelRepo.CreateDefaultChannel(ctx, ws2.ID, hank.ID)
	if err != nil {
		return fmt.Errorf("create #general in Side Project: %w", err)
	}
	for _, u := range []*user.User{alice, bob, carol} {
		if _, err := channelRepo.AddMember(ctx, u.ID, spGeneral.ID, &posterRole); err != nil {
			return fmt.Errorf("add %s to Side Project #general: %w", u.DisplayName, err)
		}
	}

	spDesign := &channel.Channel{
		WorkspaceID: ws2.ID,
		Name:        "design",
		Type:        channel.TypePublic,
	}
	if err := channelRepo.Create(ctx, spDesign, hank.ID); err != nil {
		return fmt.Errorf("create #design in Side Project: %w", err)
	}
	if _, err := channelRepo.AddMember(ctx, alice.ID, spDesign.ID, &posterRole); err != nil {
		return fmt.Errorf("add Alice to #design: %w", err)
	}

	slog.Info("created Side Project channels")

	// --- Hand-written messages ---
	handwritten := []struct {
		channelID string
		userID    string
		content   string
	}{
		{acmeGeneral.ID, alice.ID, "Welcome to Acme Corp! Glad to have everyone here."},
		{acmeGeneral.ID, bob.ID, "Thanks Alice! Excited to get started."},
		{acmeGeneral.ID, carol.ID, "Hey everyone! Looking forward to working together."},
		{acmeGeneral.ID, dave.ID, "Hello from the engineering team!"},
		{acmeGeneral.ID, eve.ID, "Hi all! Quick question — where do we track sprint tasks?"},
		{acmeGeneral.ID, alice.ID, "We use Linear for sprint planning. I'll send you an invite."},
		{acmeGeneral.ID, frank.ID, "Just joined! What's the WiFi password?"},
		{acmeGeneral.ID, grace.ID, "Hi! I'm here as a contractor for the Q1 project."},
		{acmeRandom.ID, bob.ID, "Anyone up for lunch today?"},
		{acmeRandom.ID, carol.ID, "Sure! How about that new Thai place?"},
		{acmeRandom.ID, dave.ID, "I'm in. 12:30 work for everyone?"},
		{acmeRandom.ID, eve.ID, "Count me in!"},
		{adminsOnly.ID, alice.ID, "Heads up — we need to finalize the budget by Friday."},
		{adminsOnly.ID, bob.ID, "Got it. I'll have the numbers ready by Thursday."},
		{dmAliceBob.ID, alice.ID, "Hey Bob, can you review the PR I just opened?"},
		{dmAliceBob.ID, bob.ID, "Sure, I'll take a look this afternoon."},
		{dmCarolDave.ID, carol.ID, "Dave, did you see the failing test in CI?"},
		{dmCarolDave.ID, dave.ID, "Yeah, looking into it now. Seems like a flaky test."},
		{gdmAliceCarolEve.ID, alice.ID, "Let's sync on the design review tomorrow at 2pm."},
		{gdmAliceCarolEve.ID, carol.ID, "Works for me!"},
		{gdmAliceCarolEve.ID, eve.ID, "Same here. I'll prep the mockups."},
		{spGeneral.ID, hank.ID, "Welcome to the Side Project workspace!"},
		{spGeneral.ID, alice.ID, "Thanks for setting this up, Hank."},
		{spDesign.ID, hank.ID, "Sharing some initial design ideas here."},
		{spDesign.ID, alice.ID, "These look great! Love the color palette."},
	}

	handwrittenMsgs := make([]*message.Message, 0, len(handwritten))
	for _, hw := range handwritten {
		msg := &message.Message{
			ChannelID: hw.channelID,
			UserID:    &hw.userID,
			Content:   hw.content,
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			return fmt.Errorf("create handwritten message: %w", err)
		}
		handwrittenMsgs = append(handwrittenMsgs, msg)
	}

	// Add a thread to one of the handwritten messages (Eve's question)
	threadParent := handwrittenMsgs[4] // Eve's "Quick question" message
	threadReplies := []struct {
		userID  string
		content string
	}{
		{alice.ID, "Linear is at linear.app — I'll send you an invite right now."},
		{dave.ID, "We also have a shared Google doc for longer-term roadmap items."},
		{eve.ID, "Perfect, thanks both!"},
	}
	for _, tr := range threadReplies {
		msg := &message.Message{
			ChannelID:      threadParent.ChannelID,
			UserID:         &tr.userID,
			Content:        tr.content,
			ThreadParentID: &threadParent.ID,
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			return fmt.Errorf("create thread reply: %w", err)
		}
	}

	// Add reactions to some handwritten messages
	reactions := []struct {
		msgIdx int
		userID string
		emoji  string
	}{
		{0, bob.ID, ":wave:"},
		{0, carol.ID, ":wave:"},
		{0, dave.ID, ":tada:"},
		{1, alice.ID, ":thumbsup:"},
		{7, alice.ID, ":wave:"},
		{8, carol.ID, ":thumbsup:"},
		{8, dave.ID, ":thumbsup:"},
	}
	for _, r := range reactions {
		if _, err := messageRepo.AddReaction(ctx, handwrittenMsgs[r.msgIdx].ID, r.userID, r.emoji); err != nil {
			return fmt.Errorf("add reaction: %w", err)
		}
	}

	slog.Info("created handwritten messages with threads and reactions")

	// --- Generated messages ---
	rng := rand.New(rand.NewSource(42))

	// Channel configs for bulk generation
	type channelConfig struct {
		channelID string
		memberIDs []string
		count     int
	}

	acmeAllIDs := []string{alice.ID, bob.ID, carol.ID, dave.ID, eve.ID, frank.ID, grace.ID, hank.ID}
	configs := []channelConfig{
		{acmeGeneral.ID, acmeAllIDs, 1500},
		{acmeRandom.ID, acmeAllIDs, 1200},
		{adminsOnly.ID, []string{alice.ID, bob.ID}, 200},
		{dmAliceBob.ID, []string{alice.ID, bob.ID}, 200},
		{dmCarolDave.ID, []string{carol.ID, dave.ID}, 200},
		{gdmAliceCarolEve.ID, []string{alice.ID, carol.ID, eve.ID}, 200},
		{spGeneral.ID, []string{hank.ID, alice.ID, bob.ID, carol.ID}, 300},
		{spDesign.ID, []string{hank.ID, alice.ID}, 100},
	}

	totalGenerated := 0
	for _, cfg := range configs {
		if err := generateMessages(ctx, rng, messageRepo, cfg.channelID, cfg.memberIDs, cfg.count, &totalGenerated); err != nil {
			return fmt.Errorf("generate messages for channel: %w", err)
		}
	}

	slog.Info("generated messages", "total", totalGenerated)

	slog.Info("database seeded successfully")
	return nil
}

var messageTemplates = []string{
	"Just pushed the latest changes to the feature branch",
	"Can someone review my PR when they get a chance?",
	"The deployment went smoothly — no issues so far",
	"I'm going to refactor the auth module this afternoon",
	"Has anyone seen the latest metrics dashboard?",
	"Meeting notes from today's standup are in the shared doc",
	"Quick reminder: retrospective is at 3pm today",
	"The CI pipeline is green again after that fix",
	"Working on the database migration for the new schema",
	"Does anyone have experience with WebSocket reconnection handling?",
	"I'll be out of office tomorrow, back on Thursday",
	"The new feature flag system is ready for testing",
	"Found a bug in the notification service — investigating now",
	"Great progress on the sprint goals this week!",
	"Can we move the design review to 2pm instead?",
	"Just finished the API documentation for the new endpoints",
	"The load testing results look promising",
	"Heads up: we're upgrading the database this weekend",
	"I've added error handling for the edge cases we discussed",
	"The mobile app is crashing on Android 12 — anyone else seeing this?",
	"Ship it! LGTM on the latest round of changes",
	"I'm pairing with Dave on the search implementation",
	"The A/B test results are in — variant B wins by 15%",
	"Reminder to update your dependencies before the release",
	"The staging environment is ready for QA testing",
	"I've set up monitoring alerts for the new service",
	"Can we schedule a quick sync about the roadmap?",
	"The code coverage went up to 87% after the new tests",
	"Working on fixing the memory leak in the worker process",
	"The new onboarding flow is looking really clean",
	"Just merged the accessibility improvements",
	"Anyone available for a quick code review?",
	"The cache hit rate improved after the optimization",
	"I'm investigating the slow query on the reports page",
	"Updated the README with the new setup instructions",
	"The integration tests are passing locally but failing in CI",
	"Let's discuss the architecture for the real-time features",
	"I've created tickets for all the bugs from the QA round",
	"The new search indexing is about 3x faster",
	"Deployed the hotfix to production — monitoring now",
	"Can someone help me debug this CORS issue?",
	"The design system components are now published to npm",
	"I'm working on the data export feature this sprint",
	"The response time dropped to under 200ms after the optimization",
	"Has anyone tested the new SSO integration?",
	"I'll prepare the release notes for v2.1",
	"The backup system ran successfully overnight",
	"Found some interesting patterns in the error logs",
	"The team velocity has been really consistent this quarter",
	"I'm adding rate limiting to the public API endpoints",
	"The dark mode implementation is almost complete",
	"We need to update the SSL certificates next month",
	"The new caching layer reduced database load by 40%",
	"Just finished writing the migration guide for v3",
	"Anyone interested in doing a tech talk on our architecture?",
	"The performance profiling revealed some interesting bottlenecks",
	"I've set up the new logging infrastructure",
	"The user feedback from the beta has been really positive",
	"Working on improving the error messages in the CLI",
	"The API rate limits are now configurable per tenant",
	"I'll be reviewing PRs all morning — send them my way",
	"The new webhooks feature is ready for beta testing",
	"Just completed the security audit checklist",
	"The email notification templates have been updated",
	"Can we add a health check endpoint to the service?",
	"The database indexes improved query performance significantly",
	"I'm refactoring the middleware chain for better composability",
	"The feature is behind a flag until we finish testing",
	"Released the patch for the timezone handling bug",
	"The documentation site has been deployed with the latest content",
	"I'm setting up the CI pipeline for the new microservice",
	"The data migration completed with zero downtime",
	"Anyone have recommendations for a good profiling tool?",
	"The new GraphQL endpoint is ready for review",
	"I've updated the docker-compose config for local dev",
	"The user session management has been completely rewritten",
	"Let's review the sprint backlog in tomorrow's meeting",
	"The automated testing suite caught a regression early",
	"I'm adding pagination to the activity feed endpoint",
	"The new file upload service handles chunked uploads now",
	"Quick question about the error handling convention we use",
	"The monitoring dashboard shows everything is stable",
	"I've fixed the race condition in the queue processor",
	"The new permissions model is much more flexible",
	"Can we get a dev environment set up for the new hire?",
	"The API versioning strategy has been documented",
	"Just finished the POC for real-time collaboration",
	"The test suite now runs in under 2 minutes",
	"I'm adding structured logging to all the services",
	"The deployment automation saved us hours this release",
	"Anyone know why the linter is flagging this pattern?",
	"The new table component supports sorting and filtering",
	"I've created a runbook for the incident response process",
	"The type safety improvements caught several potential bugs",
	"Working on the offline support for the mobile app",
	"The CDN configuration has been optimized for better caching",
	"I'll share the architecture decision record by EOD",
	"The code review guidelines have been updated on the wiki",
	"Just wrapped up the performance testing for the release",
	"The new auth flow supports magic link login now",
	"I'm cleaning up the tech debt from last quarter",
	"The observability stack is fully operational",
}

var reactionEmojis = []string{":thumbsup:", ":heart:", ":wave:", ":fire:", ":eyes:", ":tada:", ":rocket:", ":100:", ":thinking:", ":raised_hands:"}

func generateMessages(ctx context.Context, rng *rand.Rand, messageRepo *message.Repository, channelID string, memberIDs []string, count int, totalGenerated *int) error {
	var recentMsgIDs []string // for threading

	for i := 0; i < count; i++ {
		userID := memberIDs[rng.Intn(len(memberIDs))]
		content := messageTemplates[rng.Intn(len(messageTemplates))]

		msg := &message.Message{
			ChannelID: channelID,
			UserID:    &userID,
			Content:   content,
		}

		// ~10% of messages include a @mention
		if rng.Intn(10) == 0 && len(memberIDs) > 1 {
			mentionTarget := memberIDs[rng.Intn(len(memberIDs))]
			if mentionTarget != userID {
				msg.Content = fmt.Sprintf("%s (cc <@%s>)", content, mentionTarget)
				msg.Mentions = []string{mentionTarget}
			}
		}

		// ~5% of messages are threaded replies to recent messages
		if rng.Intn(20) == 0 && len(recentMsgIDs) > 0 {
			parentID := recentMsgIDs[rng.Intn(len(recentMsgIDs))]
			msg.ThreadParentID = &parentID
		}

		if err := messageRepo.Create(ctx, msg); err != nil {
			return err
		}

		// Track recent messages as potential thread parents (keep last 50)
		if msg.ThreadParentID == nil {
			recentMsgIDs = append(recentMsgIDs, msg.ID)
			if len(recentMsgIDs) > 50 {
				recentMsgIDs = recentMsgIDs[1:]
			}
		}

		// ~10% of messages get 1-3 reactions
		if rng.Intn(10) == 0 {
			numReactions := rng.Intn(3) + 1
			for r := 0; r < numReactions; r++ {
				reactorID := memberIDs[rng.Intn(len(memberIDs))]
				emojiStr := reactionEmojis[rng.Intn(len(reactionEmojis))]
				// Ignore duplicate reaction errors
				_, _ = messageRepo.AddReaction(ctx, msg.ID, reactorID, emojiStr)
			}
		}

		*totalGenerated++

		// Sleep every 50 messages to ensure ULID ordering
		if *totalGenerated%50 == 0 {
			time.Sleep(time.Millisecond)
		}

		// Log progress every 500 messages
		if *totalGenerated%500 == 0 {
			slog.Info("seed progress", "messages_created", *totalGenerated)
		}
	}

	return nil
}
