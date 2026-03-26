package handler

import (
	"testing"
)

func TestUnauthorizedResponse(t *testing.T) {
	resp := unauthorizedResponse()
	if resp.Error.Code != ErrCodeNotAuthenticated {
		t.Errorf("expected code %q, got %q", ErrCodeNotAuthenticated, resp.Error.Code)
	}
	if resp.Error.Message != "Not authenticated" {
		t.Errorf("expected message %q, got %q", "Not authenticated", resp.Error.Message)
	}
}

func TestForbiddenResponse(t *testing.T) {
	resp := forbiddenResponse("Permission denied")
	if resp.Error.Code != ErrCodePermissionDenied {
		t.Errorf("expected code %q, got %q", ErrCodePermissionDenied, resp.Error.Code)
	}
	if resp.Error.Message != "Permission denied" {
		t.Errorf("expected message %q, got %q", "Permission denied", resp.Error.Message)
	}
}

func TestNotFoundResponse(t *testing.T) {
	resp := notFoundResponse("Channel not found")
	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected code %q, got %q", ErrCodeNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "Channel not found" {
		t.Errorf("expected message %q, got %q", "Channel not found", resp.Error.Message)
	}
}

func TestBadRequestResponse(t *testing.T) {
	resp := badRequestResponse(ErrCodeValidationError, "Name is required")
	if resp.Error.Code != ErrCodeValidationError {
		t.Errorf("expected code %q, got %q", ErrCodeValidationError, resp.Error.Code)
	}
	if resp.Error.Message != "Name is required" {
		t.Errorf("expected message %q, got %q", "Name is required", resp.Error.Message)
	}
}

func TestNotAMemberResponse(t *testing.T) {
	resp := notAMemberResponse("Not a member of this workspace")
	if resp.Error.Code != ErrCodeNotAMember {
		t.Errorf("expected code %q, got %q", ErrCodeNotAMember, resp.Error.Code)
	}
	if resp.Error.Message != "Not a member of this workspace" {
		t.Errorf("expected message %q, got %q", "Not a member of this workspace", resp.Error.Message)
	}
}
