package models

// Эти структуры не используются напрямую, но отражают таблицы в БД
// Логика реакций обрабатывается через SQL-запросы в обработчиках
type PostReaction struct {
	ID      int
	PostID  int
	UserID  int
	IsLike  bool // true for like, false for dislike
	Dislike bool // true for dislike, false for like (synced with is_like)
}

type CommentReaction struct {
	ID        int
	CommentID int
	UserID    int
	IsLike    bool // true for like, false for dislike
	Dislike   bool // true for dislike, false for like (synced with is_like)
}
