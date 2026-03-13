package types

type AnswerDetails struct {
	MaxChars int `bson:"max_chars,omitempty" json:"max_chars,omitempty"`
}

type FormAnswer struct {
	Value any `bson:"value,omitempty" json:"value,omitempty"` // can be string, []string, number, boolean, or nil
}

type FormQuestion struct {
	ID            string         `bson:"id" json:"id"`
	Type          string         `bson:"type" json:"type"` // textbox, multiple_choice, dropdown, checkbox
	Question      string         `bson:"question" json:"question"`
	Required      bool           `bson:"required" json:"required"`
	AnswerDetails *AnswerDetails `bson:"answer_details,omitempty" json:"answer_details,omitempty"`
	AnswerOptions []string       `bson:"answer_options,omitempty" json:"answer_options,omitempty"`
}

type Event struct {
	ID               string         `bson:"_id" json:"id"`
	Name             string         `bson:"name" json:"name"`
	Date             string         `bson:"date" json:"date"`
	Time             string         `bson:"time" json:"time"`
	Location         string         `bson:"location" json:"location"`
	Description      string         `bson:"description" json:"description"`
	Admins           []string       `bson:"admins" json:"admins"`
	RegistrationForm []FormQuestion `bson:"registration_form" json:"registration_form"`
	MaxAttendees     int            `bson:"max_attendees" json:"max_attendees"`
	CreatedAt        string         `bson:"created_at" json:"created_at"`
	Status           string         `bson:"status" json:"status"` // draft, published, closed
}
