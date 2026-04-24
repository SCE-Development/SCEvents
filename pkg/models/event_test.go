package models

import "testing"

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name   string
		admins []string
		userID string
		want   bool
	}{
		{
			name:   "user is an admin",
			admins: []string{"user-1", "user-2"},
			userID: "user-1",
			want:   true,
		},
		{
			name:   "user is not an admin",
			admins: []string{"user-1", "user-2"},
			userID: "user-3",
			want:   false,
		},
		{
			name:   "no admins",
			admins: []string{},
			userID: "user-1",
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev := Event{Admins: tc.admins}
			if got := ev.IsAdmin(tc.userID); got != tc.want {
				t.Errorf("IsAdmin(%q) = %v, want %v", tc.userID, got, tc.want)
			}
		})
	}
}
