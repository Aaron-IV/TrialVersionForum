package models

import "time"

type Post struct {
	ID           int
	UserID       int
	Title        string
	Content      string
	CreatedAt    time.Time
	Author       string     // Username of the author
	Categories   []Category // Categories associated with this post
	Likes        int
	Dislikes     int
	UserReaction int // 1 for like, -1 for dislike, 0 for no reaction by current user
	CommentCount int
}
