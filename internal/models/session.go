package models

import "time"

type Session struct {
	ID      int
	UserID  int
	UUID    string
	Expires time.Time
}
