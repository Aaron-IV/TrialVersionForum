package models

import "time"

type Comment struct {
	ID           int
	PostID       int
	UserID       int
	Content      string
	CreatedAt    time.Time
	Author       string // Username of the author
	Likes        int
	Dislikes     int
	UserReaction int // 1 for like, -1 for dislike, 0 for no reaction by current user
}
