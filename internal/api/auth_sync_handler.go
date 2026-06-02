package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/models"
	"github.com/clerk/clerk-sdk-go/v2/user"
)

type AuthSyncHandler struct {
	userRepo models.UserRepository
	orgRepo  models.OrgRepository
	logger   *slog.Logger
}

func NewAuthSyncHandler(logger *slog.Logger, userRepo models.UserRepository, orgRepo models.OrgRepository) *AuthSyncHandler {
	return &AuthSyncHandler{userRepo: userRepo, orgRepo: orgRepo, logger: logger}
}

func (h *AuthSyncHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/auth/me", AuthMiddleware(h.logger, h.userRepo)(http.HandlerFunc(h.handleAuthSync)))
}

func (h *AuthSyncHandler) handleAuthSync(w http.ResponseWriter, r *http.Request) {
	clerkID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || clerkID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Check if user exists in PostgreSQL
	dbUser, err := h.userRepo.GetByClerkID(clerkID)
	if err == nil && dbUser != nil {
		// User exists!
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dbUser)
		return
	}

	// Fetch email from Clerk
	email := clerkID + "@clerk.local" // fallback
	clerkUser, clerkErr := user.Get(r.Context(), clerkID)
	if clerkErr == nil && clerkUser != nil && len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	} else if clerkErr != nil {
		h.logger.Warn("Failed to fetch user from Clerk API", "error", clerkErr)
	}

	// 2. Check if user has a pending invite (exists by email but no clerk_id)
	invitedUser, err := h.userRepo.GetByEmail(email)
	if err == nil && invitedUser != nil {
		h.logger.Info("Found pending invite for user, linking Clerk ID", "email", email, "clerk_id", clerkID)
		if updateErr := h.userRepo.UpdateClerkID(invitedUser.ID, clerkID); updateErr != nil {
			h.logger.Error("Failed to link clerk_id to invited user", "error", updateErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		invitedUser.ClerkID = clerkID
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(invitedUser)
		return
	}

	// 3. User does not exist at all. Auto-provision!
	h.logger.Info("Provisioning new user and organization", "clerk_id", clerkID, "email", email)

	// Create Organization
	newOrg := &models.Organization{
		Name: "Personal Workspace",
		Plan: "startup",
	}
	if err := h.orgRepo.Create(newOrg); err != nil {
		h.logger.Error("Failed to auto-create organization", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create User linked to Org
	newUser := &models.User{
		OrgID:   newOrg.ID,
		ClerkID: clerkID,
		Email:   email,
		Role:    "admin",
	}
	if err := h.userRepo.Create(newUser); err != nil {
		h.logger.Error("Failed to auto-create user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newUser)
}
