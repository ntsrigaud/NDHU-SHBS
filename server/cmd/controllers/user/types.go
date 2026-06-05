package user

import "strings"

// UpdateMeRequest allows partial updates to the authenticated user's profile.
//
// If avatar_image_id is provided as an empty string, the avatar is cleared.
type UpdateMeRequest struct {
	Name          *string `json:"name"`
	AvatarImageID *string `json:"avatar_image_id"`
}

func normalizeUpdateMeRequest(r *UpdateMeRequest) {
	if r.Name != nil {
		n := strings.TrimSpace(*r.Name)
		r.Name = &n
	}
	if r.AvatarImageID != nil {
		a := strings.TrimSpace(*r.AvatarImageID)
		r.AvatarImageID = &a
	}
}
