package models

import (
	"reflect"
	"testing"
	"time"
)

func TestApplyDefaults(t *testing.T) {
	e := &Event{}
	e.ApplyDefaults()

	if e.Status != StatusDraft {
		t.Errorf("expected StatusDraft, got %s", e.Status)
	}
	if e.Visibility != VisibilityPublic {
		t.Errorf("expected VisibilityPublic, got %s", e.Visibility)
	}
}

func TestNormalize(t *testing.T) {
	e := &Event{
		Status:             " " + StatusPublished + " ",
		Visibility:         VisibilityPublic,
		MinimumVisibleRole: RoleMember,
		WaitlistEnabled:    false,
		WaitlistSize:       10,
	}
	e.normalize()

	if e.Status != StatusPublished {
		t.Errorf("expected %s, got %s", StatusPublished, e.Status)
	}
	if e.MinimumVisibleRole != "" {
		t.Errorf("expected empty MinimumVisibleRole, got %s", e.MinimumVisibleRole)
	}
	if e.WaitlistSize != 0 {
		t.Errorf("expected WaitlistSize to be 0, got %d", e.WaitlistSize)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid public event",
			event: Event{
				Name:         "Test",
				Date:         "2026-05-01",
				Time:         "10:00",
				Location:     "Room 101",
				Status:       StatusPublished,
				Visibility:   VisibilityPublic,
				MaxAttendees: 50,
				Admins:       []string{"admin-1"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			event: Event{
				Date:         "2026-05-01",
				Time:         "10:00",
				Location:     "Room 101",
				Status:       StatusPublished,
				Visibility:   VisibilityPublic,
				MaxAttendees: 50,
			},
			wantErr: true,
		},
		{
			name: "invalid waitlist size",
			event: Event{
				Name:            "Test",
				Date:            "2026-05-01",
				Time:            "10:00",
				Location:        "Room 101",
				Status:          StatusPublished,
				Visibility:      VisibilityPublic,
				WaitlistEnabled: true,
				WaitlistSize:    0,
				MaxAttendees:    50,
			},
			wantErr: true,
		},
		{
			name: "private event without minimum role",
			event: Event{
				Name:         "Test",
				Date:         "2026-05-01",
				Time:         "10:00",
				Location:     "Room 101",
				Status:       StatusPublished,
				Visibility:   VisibilityPrivate,
				MaxAttendees: 50,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_Validate_MaxAttendees(t *testing.T) {
	// Setup a valid base event to isolate MaxAttendees validation
	baseEvent := func() Event {
		return Event{
			Name:       "Test Event",
			Date:       "2026-05-01",
			Time:       "10:00",
			Location:   "Room 101",
			Status:     StatusDraft,
			Visibility: VisibilityPublic,
			Admins:     []string{"admin-1"},
		}
	}

	tests := []struct {
		name         string
		maxAttendees int
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "valid positive max attendees",
			maxAttendees: 50,
			wantErr:      false,
		},
		{
			name:         "invalid zero max attendees",
			maxAttendees: 0,
			wantErr:      true,
			errMessage:   "max_attendees must be greater than 0, or -1 for no limit",
		},
		{
			name:         "valid unlimited max attendees",
			maxAttendees: -1,
			wantErr:      false,
		},
		{
			name:         "invalid negative max attendees",
			maxAttendees: -5,
			wantErr:      true,
			errMessage:   "max_attendees must be greater than 0, or -1 for no limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := baseEvent()
			ev.MaxAttendees = tt.maxAttendees

			err := ev.Validate()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if err.Error() != tt.errMessage {
					t.Errorf("expected error message %q, got %q", tt.errMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsListedAdmin(t *testing.T) {
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
			if got := ev.IsListedAdmin(tc.userID); got != tc.want {
				t.Errorf("IsListedAdmin(%q) = %v, want %v", tc.userID, got, tc.want)
			}
		})
	}
}

func TestCanEdit(t *testing.T) {
	e := &Event{
		Admins: []string{"user1"},
	}
	if !e.CanEdit("user1", RoleMember) {
		t.Errorf("expected true for user1")
	}
	if e.CanEdit("user2", RoleAdmin) {
		t.Errorf("expected false for user2 because they are not in admins even if they are site admin when admins list is not empty")
	}

	eEmptyAdmins := &Event{}
	if !eEmptyAdmins.CanEdit("user2", RoleAdmin) {
		t.Errorf("expected true for user2 with admin site role when admins list is empty")
	}
	if eEmptyAdmins.CanEdit("user2", RoleMember) {
		t.Errorf("expected false for user2 with member site role when admins list is empty")
	}
}

func TestValidateRegistration(t *testing.T) {
	e := &Event{
		RegistrationForm: []FormQuestion{
			{ID: "q1", Required: true},
			{ID: "q2", Required: false},
		},
	}

	answersValid := map[string]any{"q1": "answer1"}
	if err := e.ValidateRegistration(answersValid); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	answersMissing := map[string]any{"q2": "answer2"}
	if err := e.ValidateRegistration(answersMissing); err == nil {
		t.Errorf("expected error for missing required answer")
	}

	answersEmpty := map[string]any{"q1": "   "}
	if err := e.ValidateRegistration(answersEmpty); err == nil {
		t.Errorf("expected error for empty required answer")
	}
}

func TestSanitizeUpdateFields(t *testing.T) {
	fields := map[string]interface{}{
		"id":           "123",
		"_id":          "123",
		"created_at":   "now",
		"published_at": "later",
		"name":         "new name",
	}

	SanitizeUpdateFields(fields)

	if _, ok := fields["id"]; ok {
		t.Errorf("expected id to be removed")
	}
	if _, ok := fields["_id"]; ok {
		t.Errorf("expected _id to be removed")
	}
	if _, ok := fields["created_at"]; ok {
		t.Errorf("expected created_at to be removed")
	}
	if _, ok := fields["name"]; !ok {
		t.Errorf("expected name to be kept")
	}
	if _, ok := fields["published_at"]; ok {
		t.Errorf("expected published_at to be removed")
	}
}

func TestValidate_EndDate(t *testing.T) {
	base := func() Event {
		return Event{
			Name:         "Test Event",
			Date:         "2026-05-01",
			Time:         "10:00",
			Location:     "Room 101",
			Status:       StatusDraft,
			Visibility:   VisibilityPublic,
			MaxAttendees: 10,
			Admins:       []string{"admin-1"},
		}
	}

	tests := []struct {
		name    string
		endDate string
		wantErr string
	}{
		{
			name:    "no end_date is valid",
			endDate: "",
			wantErr: "",
		},
		{
			name:    "end_date equal to date is valid",
			endDate: "2026-05-01",
			wantErr: "",
		},
		{
			name:    "end_date after date is valid",
			endDate: "2026-05-02",
			wantErr: "",
		},
		{
			name:    "end_date before date returns error",
			endDate: "2026-04-30",
			wantErr: "end_date must be on or after start date",
		},
		{
			name:    "invalid end_date format returns error",
			endDate: "05-01-2026",
			wantErr: "end_date is not a valid date (expected YYYY-MM-DD)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev := base()
			ev.EndDate = tc.endDate

			err := ev.Validate()

			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error %q, got nil", tc.wantErr)
				return
			}
			if err.Error() != tc.wantErr {
				t.Errorf("expected error %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestApplyPatch(t *testing.T) {
	e := &Event{
		Name: "Old Name",
	}

	fields := map[string]interface{}{
		"name":             "New Name",
		"max_attendees":    float64(100),
		"waitlist_enabled": true,
		"waitlist_size":    float64(50),
	}

	if err := e.ApplyPatch(fields); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if e.Name != "New Name" {
		t.Errorf("expected name to be New Name, got %s", e.Name)
	}
	if e.MaxAttendees != 100 {
		t.Errorf("expected max_attendees to be 100, got %d", e.MaxAttendees)
	}
	if !e.WaitlistEnabled {
		t.Errorf("expected waitlist_enabled to be true")
	}
	if e.WaitlistSize != 50 {
		t.Errorf("expected waitlist_size to be 50, got %d", e.WaitlistSize)
	}

	invalidFields := map[string]interface{}{
		"name": 123,
	}
	if err := e.ApplyPatch(invalidFields); err == nil {
		t.Errorf("expected error for invalid type")
	}
}

func TestApplyPatch_Admins(t *testing.T) {
	e := &Event{
		Admins: []string{"admin-old"},
	}

	fields := map[string]interface{}{
		"admins": []interface{}{" admin-1 ", "admin-2", "admin-1"},
	}

	if err := e.ApplyPatch(fields); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expected := []string{"admin-1", "admin-2"}
	if !reflect.DeepEqual(e.Admins, expected) {
		t.Errorf("expected admins %v, got %v", expected, e.Admins)
	}
}

func TestApplyPatch_AdminsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]interface{}
		wantErr string
	}{
		{
			name: "not an array",
			fields: map[string]interface{}{
				"admins": "admin-1",
			},
			wantErr: "admins must be an array",
		},
		{
			name: "blank admin",
			fields: map[string]interface{}{
				"admins": []interface{}{"admin-1", " "},
			},
			wantErr: "admins[1] cannot be empty",
		},
		{
			name: "non-string admin",
			fields: map[string]interface{}{
				"admins": []interface{}{"admin-1", float64(2)},
			},
			wantErr: "admins[1] must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{Admins: []string{"admin-old"}}
			err := e.ApplyPatch(tt.fields)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestApplyPatch_RegistrationForm(t *testing.T) {
	e := &Event{
		Name: "Test Event",
	}

	fields := map[string]interface{}{
		"registration_form": []interface{}{
			map[string]interface{}{
				"id":       "q1",
				"type":     "textbox",
				"question": "What is your name?",
				"required": true,
				"answer_details": map[string]interface{}{
					"max_chars": float64(200),
				},
			},
			map[string]interface{}{
				"id":             "q2",
				"type":           "multiple_choice",
				"question":       "Favorite color?",
				"required":       false,
				"answer_options": []interface{}{"Red", "Blue", "Green"},
			},
		},
	}

	if err := e.ApplyPatch(fields); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(e.RegistrationForm) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(e.RegistrationForm))
	}

	q1 := e.RegistrationForm[0]
	if q1.ID != "q1" || q1.Type != "textbox" || q1.Question != "What is your name?" || !q1.Required {
		t.Errorf("q1 fields mismatch: %+v", q1)
	}
	if q1.AnswerDetails == nil || q1.AnswerDetails.MaxChars != 200 {
		t.Errorf("q1 answer_details mismatch: %+v", q1.AnswerDetails)
	}

	q2 := e.RegistrationForm[1]
	if q2.ID != "q2" || len(q2.AnswerOptions) != 3 {
		t.Errorf("q2 fields mismatch: %+v", q2)
	}

	// Test invalid type
	invalidFields := map[string]interface{}{
		"registration_form": "not an array",
	}
	if err := e.ApplyPatch(invalidFields); err == nil {
		t.Errorf("expected error for invalid registration_form type")
	}
}

func TestCanView(t *testing.T) {
	base := Event{
		ID:                 "event-1",
		Name:               "Test Event",
		Status:             StatusPublished,
		Visibility:         VisibilityPublic,
		MinimumVisibleRole: "",
	}

	tests := []struct {
		name   string
		event  Event
		viewer EventViewer
		want   bool
	}{
		{
			name:   "site admin can view anything",
			event:  Event{Status: StatusDraft},
			viewer: EventViewer{UserID: "admin-1", AccessLevel: 3},
			want:   true,
		},
		{
			name: "listed event admin can view own draft",
			event: Event{
				Status: StatusDraft,
				Admins: []string{"user-1"},
			},
			viewer: EventViewer{UserID: "user-1", AccessLevel: 1},
			want:   true,
		},
		{
			name:   "non-admin cannot view draft",
			event:  Event{Status: StatusDraft},
			viewer: EventViewer{UserID: "user-2", AccessLevel: 1},
			want:   false,
		},
		{
			name:   "published public visible to anonymous",
			event:  base,
			viewer: EventViewer{},
			want:   true,
		},
		{
			name: "published private member visible to member",
			event: Event{
				Status:             StatusPublished,
				Visibility:         VisibilityPrivate,
				MinimumVisibleRole: RoleMember,
			},
			viewer: EventViewer{AccessLevel: 1},
			want:   true,
		},
		{
			name: "published private officer not visible to member",
			event: Event{
				Status:             StatusPublished,
				Visibility:         VisibilityPrivate,
				MinimumVisibleRole: RoleOfficer,
			},
			viewer: EventViewer{AccessLevel: 1},
			want:   false,
		},
		{
			name: "published private officer visible to officer",
			event: Event{
				Status:             StatusPublished,
				Visibility:         VisibilityPrivate,
				MinimumVisibleRole: RoleOfficer,
			},
			viewer: EventViewer{AccessLevel: 2},
			want:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.CanView(tc.viewer); got != tc.want {
				t.Fatalf("CanView() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldAutoPublish(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name  string
		event Event
		want  bool
	}{
		{
			name:  "nil publish date does not auto publish",
			event: Event{Status: StatusDraft},
			want:  false,
		},
		{
			name: "future publish date does not auto publish",
			event: Event{
				Status:      StatusDraft,
				PublishDate: &future,
			},
			want: false,
		},
		{
			name: "past publish date auto publishes",
			event: Event{
				Status:      StatusDraft,
				PublishDate: &past,
			},
			want: true,
		},
		{
			name: "published event does not auto publish again",
			event: Event{
				Status:      StatusPublished,
				PublishDate: &past,
			},
			want: false,
		},
		{
			name: "closed event does not auto publish",
			event: Event{
				Status:      StatusClosed,
				PublishDate: &past,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.ShouldAutoPublish(now); got != tc.want {
				t.Fatalf("ShouldAutoPublish() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSyncPublicationState(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)

	t.Run("manual publish sets timestamps when missing", func(t *testing.T) {
		ev := Event{Status: StatusPublished}

		ev.SyncPublicationState(now)

		if ev.PublishDate == nil {
			t.Fatal("expected PublishDate to be set")
		}
		if ev.PublishedAt == nil {
			t.Fatal("expected PublishedAt to be set")
		}
	})

	t.Run("due draft auto publishes", func(t *testing.T) {
		ev := Event{
			Status:      StatusDraft,
			PublishDate: &past,
		}

		ev.SyncPublicationState(now)

		if ev.Status != StatusPublished {
			t.Fatalf("expected status published, got %s", ev.Status)
		}
		if ev.PublishedAt == nil {
			t.Fatal("expected PublishedAt to be set")
		}
	})

	t.Run("future draft stays draft", func(t *testing.T) {
		future := now.Add(time.Hour)
		ev := Event{
			Status:      StatusDraft,
			PublishDate: &future,
		}

		ev.SyncPublicationState(now)

		if ev.Status != StatusDraft {
			t.Fatalf("expected status draft, got %s", ev.Status)
		}
		if ev.PublishedAt != nil {
			t.Fatal("expected PublishedAt to remain nil")
		}
	})
}

func TestValidate_PublishDateRules(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	t.Run("closed event cannot have publish date", func(t *testing.T) {
		ev := Event{
			Name:         "Test",
			Date:         "2026-05-01",
			Time:         "10:00",
			Location:     "Room 101",
			Status:       StatusClosed,
			Visibility:   VisibilityPublic,
			MaxAttendees: 10,
			Admins:       []string{"admin-1"},
			PublishDate:  &now,
		}

		err := ev.Validate()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("draft event with publish date is valid", func(t *testing.T) {
		ev := Event{
			Name:         "Test",
			Date:         "2026-05-01",
			Time:         "10:00",
			Location:     "Room 101",
			Status:       StatusDraft,
			Visibility:   VisibilityPublic,
			MaxAttendees: 10,
			Admins:       []string{"admin-1"},
			PublishDate:  &now,
		}

		if err := ev.Validate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestApplyPatch_PublishDate(t *testing.T) {
	e := &Event{}

	t.Run("sets publish_date from RFC3339 string", func(t *testing.T) {
		fields := map[string]interface{}{
			"publish_date": "2026-05-01T12:00:00Z",
		}

		if err := e.ApplyPatch(fields); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.PublishDate == nil {
			t.Fatal("expected PublishDate to be set")
		}
	})

	t.Run("clears publish_date with nil", func(t *testing.T) {
		e.PublishDate = func() *time.Time {
			tm := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
			return &tm
		}()

		fields := map[string]interface{}{
			"publish_date": nil,
		}

		if err := e.ApplyPatch(fields); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.PublishDate != nil {
			t.Fatal("expected PublishDate to be nil")
		}
	})

	t.Run("rejects invalid publish_date type", func(t *testing.T) {
		fields := map[string]interface{}{
			"publish_date": 123,
		}

		if err := e.ApplyPatch(fields); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
