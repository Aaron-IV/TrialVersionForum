package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"forum/internal/auth"
	"forum/internal/database"
	"forum/internal/models"
)

// Global templates variable to parse templates once at startup
var templates *template.Template

// TemplateData holds data passed to HTML templates.
type TemplateData struct {
	User              *models.User
	Error             string
	Categories        []models.Category
	Posts             []models.Post
	Post              models.Post
	Comments          []models.Comment
	Comment           models.Comment // For edit_comment.html
	TotalPages        int
	Page              int
	CurrentCategoryID int
	CurrentFilter     string
}

// init (остается без изменений)
func init() {
	var err error
	templates, err = template.New("").Funcs(template.FuncMap{
		"iterate": func(count int) []int {
			var items []int
			for i := 0; i < count; i++ {
				items = append(items, i)
			}
			return items
		},
		"add": func(a, b int) int { return a + b },
		"dec": func(a int) int { return a - 1 },
		"inc": func(a int) int { return a + 1 },
		"paginationURL": func(page, categoryID int, filter string) template.URL {
			u, _ := url.Parse("/")
			q := u.Query()
			q.Set("page", strconv.Itoa(page))
			if categoryID > 0 {
				q.Set("category", strconv.Itoa(categoryID))
			}
			if filter != "" {
				q.Set("filter", filter)
			}
			u.RawQuery = q.Encode()
			return template.URL(u.String())
		},
		"formatDateTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 02, 2006 at 15:04")
		},
		"trimContent": func(s string) template.HTML {
			// Агрессивная обрезка: удаляем все пробелы и переносы строк в начале и конце
			s = strings.TrimSpace(s)
			if s == "" {
				return template.HTML("")
			}
			// Удаляем все пробелы, табы и переносы строк в начале каждой строки
			lines := strings.Split(s, "\n")
			var cleanedLines []string
			for _, line := range lines {
				// Удаляем пробелы и табы в начале и конце строки
				trimmedLine := strings.TrimSpace(line)
				// Пропускаем пустые строки
				if trimmedLine == "" {
					continue
				}
				// Экранируем HTML и добавляем строку
				escapedLine := template.HTMLEscapeString(trimmedLine)
				cleanedLines = append(cleanedLines, escapedLine)
			}
			if len(cleanedLines) == 0 {
				return template.HTML("")
			}
			// Объединяем строки через <br>
			result := strings.Join(cleanedLines, "<br>")
			return template.HTML(result)
		},
	}).ParseGlob("internal/templates/*.html")
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}
}

// renderTemplate (остается без изменений)
func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data TemplateData) {
	// Используем буфер для рендеринга, чтобы не отправлять частичный вывод при ошибке
	var buf strings.Builder
	err := templates.ExecuteTemplate(&buf, templateName, data)
	if err != nil {
		log.Printf("Error rendering template %s: %v", templateName, err)
		log.Printf("Template data: User=%v, Comment.ID=%d, Comment.Content length=%d, Post.ID=%d",
			data.User != nil, data.Comment.ID, len(data.Comment.Content), data.Post.ID)
		// Если заголовки еще не отправлены, отправляем ошибку
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	// Если рендеринг успешен, отправляем результат
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(buf.String()))
}

// HTTP Error Handlers (остаются без изменений)
func Render400(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusBadRequest)
	renderTemplate(w, r, "error.html", TemplateData{Error: "400 Bad Request: " + message})
}

func Render403(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusForbidden)
	renderTemplate(w, r, "error.html", TemplateData{Error: "403 Forbidden: " + message})
}

func Render404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, r, "error.html", TemplateData{Error: "404 Not Found"})
}

func Render405(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	renderTemplate(w, r, "error.html", TemplateData{Error: "405 Method Not Allowed"})
}

func Render500(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal Server Error: %s", message)
	w.WriteHeader(http.StatusInternalServerError)
	renderTemplate(w, r, "error.html", TemplateData{Error: "500 Internal Server Error: " + message})
}

const (
	postsPerPage  = 5
	maxTitleLen   = 256
	maxContentLen = 2500
	maxCommentLen = 500
)

// countRunes counts the number of runes (Unicode characters) in a string
func countRunes(s string) int {
	return utf8.RuneCountInString(s)
}

// HomeHandler displays the main page with posts, filtering, and pagination.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		Render404(w, r)
		return
	}

	// Проверяем метод GET
	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}

	user := auth.GetUserFromContext(r.Context())

	// ИСПРАВЛЕНИЕ: Объявляем `data` здесь один раз
	data := TemplateData{User: user}

	// Fetch categories
	rows, err := database.DB.Query("SELECT id, name FROM categories ORDER BY id")
	if err != nil {
		Render500(w, r, "Failed to load categories: "+err.Error())
		return
	}
	defer rows.Close()
	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Printf("Error scanning category: %v", err)
			continue
		}
		categories = append(categories, c)
	}
	data.Categories = categories

	// Pagination and Filtering
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	data.Page = page

	categoryID, _ := strconv.Atoi(r.URL.Query().Get("category"))
	data.CurrentCategoryID = categoryID
	filter := r.URL.Query().Get("filter")
	data.CurrentFilter = filter

	var (
		queryBuilder      strings.Builder
		countQueryBuilder strings.Builder
		args              []interface{}
		countArgs         []interface{}
		whereClauses      []string
		joinClauses       string
		countJoinClauses  string
	)

	baseSelect := `
        SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username,
               (SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count,
               (SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND is_like = 1) AS likes,
               (SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND is_like = 0) AS dislikes
        FROM posts p
        JOIN users u ON p.user_id = u.id
    `
	countBaseSelect := `SELECT COUNT(DISTINCT p.id) FROM posts p`

	if categoryID > 0 {
		joinClauses += " JOIN post_categories pc ON p.id = pc.post_id "
		countJoinClauses += " JOIN post_categories pc ON p.id = pc.post_id "
		whereClauses = append(whereClauses, "pc.category_id = ?")
		args = append(args, categoryID)
		countArgs = append(countArgs, categoryID)
	}

	if filter != "" {
		if filter == "myposts" && user != nil {
			whereClauses = append(whereClauses, "p.user_id = ?")
			args = append(args, user.ID)
			countArgs = append(countArgs, user.ID)
		} else if filter == "liked" && user != nil {
			joinClauses += " JOIN post_reactions lr ON p.id = lr.post_id "
			countJoinClauses += " JOIN post_reactions lr ON p.id = lr.post_id "
			whereClauses = append(whereClauses, "lr.user_id = ? AND lr.is_like = 1")
			args = append(args, user.ID)
			countArgs = append(countArgs, user.ID)
		} else if filter == "disliked" && user != nil {
			joinClauses += " JOIN post_reactions dr ON p.id = dr.post_id "
			countJoinClauses += " JOIN post_reactions dr ON p.id = dr.post_id "
			whereClauses = append(whereClauses, "dr.user_id = ? AND dr.is_like = 0")
			args = append(args, user.ID)
			countArgs = append(countArgs, user.ID)
		}
	}

	queryBuilder.WriteString(baseSelect)
	queryBuilder.WriteString(joinClauses)
	countQueryBuilder.WriteString(countBaseSelect)
	countQueryBuilder.WriteString(countJoinClauses)

	if len(whereClauses) > 0 {
		whereStr := " WHERE " + strings.Join(whereClauses, " AND ")
		queryBuilder.WriteString(whereStr)
		countQueryBuilder.WriteString(whereStr)
	}

	// Order by depending on filter
	switch filter {
	case "most_liked":
		queryBuilder.WriteString(" ORDER BY likes DESC, p.created_at DESC ")
	case "most_commented":
		queryBuilder.WriteString(" ORDER BY comment_count DESC, p.created_at DESC ")
	default:
		queryBuilder.WriteString(" ORDER BY p.created_at DESC ")
	}
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", postsPerPage, (page-1)*postsPerPage))

	var totalPosts int
	err = database.DB.QueryRow(countQueryBuilder.String(), countArgs...).Scan(&totalPosts)
	if err != nil {
		Render500(w, r, "Failed to count posts: "+err.Error())
		return
	}
	data.TotalPages = (totalPosts + postsPerPage - 1) / postsPerPage

	if totalPosts > 0 && page > data.TotalPages {
		http.Redirect(w, r, fmt.Sprintf("/?page=%d&category=%d&filter=%s", data.TotalPages, categoryID, filter), http.StatusFound)
		return
	}

	rows, err = database.DB.Query(queryBuilder.String(), args...)
	if err != nil {
		Render500(w, r, "Failed to load posts: "+err.Error())
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt, &p.Author, &p.CommentCount, &p.Likes, &p.Dislikes); err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}

		// Агрессивная обрезка: удаляем все пробелы и пустые строки в начале
		p.Content = strings.TrimSpace(p.Content)
		// Удаляем все пустые строки в начале
		lines := strings.Split(p.Content, "\n")
		var cleanedLines []string
		startFound := false
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			// Пропускаем пустые строки только в начале
			if !startFound && trimmedLine == "" {
				continue
			}
			startFound = true
			cleanedLines = append(cleanedLines, trimmedLine)
		}
		p.Content = strings.Join(cleanedLines, "\n")
		p.Content = strings.TrimSpace(p.Content)

		catRows, catErr := database.DB.Query(`SELECT c.id, c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?`, p.ID)
		if catErr != nil {
			log.Printf("Error fetching categories for post %d: %v", p.ID, catErr)
		} else {
			for catRows.Next() {
				var cat models.Category
				if err := catRows.Scan(&cat.ID, &cat.Name); err != nil {
					log.Printf("Error scanning post category: %v", err)
					continue
				}
				p.Categories = append(p.Categories, cat)
			}
			catRows.Close()
		}

		posts = append(posts, p)
	}
	data.Posts = posts

	log.Printf("HomeHandler: Loaded %d posts for page %d (total: %d)", len(posts), page, totalPosts)

	renderTemplate(w, r, "home_page.html", data)
}

// CreatePostHandler (остается без изменений)
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	data := TemplateData{User: user}

	if r.Method == http.MethodPost {
		if user == nil {
			log.Printf("CreatePostHandler: Unauthorized attempt to create post")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		_ = r.ParseForm()

		title := strings.TrimSpace(r.FormValue("title"))
		contentRaw := r.FormValue("content")
		// Normalize line endings to keep size consistent across browsers
		contentNorm := strings.ReplaceAll(contentRaw, "\r\n", "\n")
		// Агрессивная обрезка: удаляем все пробелы и пустые строки в начале
		contentTrim := strings.TrimSpace(contentNorm)
		// Удаляем все пустые строки в начале
		lines := strings.Split(contentTrim, "\n")
		var cleanedLines []string
		startFound := false
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			// Пропускаем пустые строки только в начале
			if !startFound && trimmedLine == "" {
				continue
			}
			startFound = true
			cleanedLines = append(cleanedLines, trimmedLine)
		}
		contentTrim = strings.Join(cleanedLines, "\n")
		// Финальная обрезка
		contentTrim = strings.TrimSpace(contentTrim)
		categoryIDs := r.Form["categories"]

		titleRuneCount := countRunes(title)
		contentRuneCount := countRunes(contentTrim)
		log.Printf("CreatePostHandler: User %d attempting to create post with title length: %d runes, content length: %d runes", user.ID, titleRuneCount, contentRuneCount)

		// Валидация входных данных (подсчет символов Unicode, а не байтов)
		if titleRuneCount == 0 || titleRuneCount > maxTitleLen {
			log.Printf("CreatePostHandler: Invalid title length: %d runes (max: %d)", titleRuneCount, maxTitleLen)
			Render400(w, r, fmt.Sprintf("Title must be between 1 and %d characters.", maxTitleLen))
			return
		}
		if contentRuneCount == 0 || contentRuneCount > maxContentLen {
			log.Printf("CreatePostHandler: Invalid content length: %d runes (max: %d)", contentRuneCount, maxContentLen)
			Render400(w, r, fmt.Sprintf("Content must be between 1 and %d characters.", maxContentLen))
			return
		}
		if len(categoryIDs) == 0 {
			log.Printf("CreatePostHandler: No categories selected")
			Render400(w, r, "At least one category must be selected.")
			return
		}

		// Validate and convert category IDs
		var validCategoryIDs []int
		for _, catIDStr := range categoryIDs {
			catID, err := strconv.Atoi(catIDStr)
			if err != nil {
				log.Printf("CreatePostHandler: Invalid category ID format: %s", catIDStr)
				Render400(w, r, "Invalid category ID.")
				return
			}
			validCategoryIDs = append(validCategoryIDs, catID)
		}

		// Create a set of unique category IDs to check for duplicates and validate
		categoryIDSet := make(map[int]bool)
		for _, id := range validCategoryIDs {
			categoryIDSet[id] = true
		}

		// Verify that all categories exist in the database
		uniqueCategoryIDs := make([]int, 0, len(categoryIDSet))
		for id := range categoryIDSet {
			uniqueCategoryIDs = append(uniqueCategoryIDs, id)
		}

		if len(uniqueCategoryIDs) == 0 {
			Render400(w, r, "At least one category must be selected.")
			return
		}

		placeholders := strings.Repeat("?,", len(uniqueCategoryIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		query := fmt.Sprintf("SELECT id FROM categories WHERE id IN (%s)", placeholders)

		args := make([]interface{}, len(uniqueCategoryIDs))
		for i, id := range uniqueCategoryIDs {
			args[i] = id
		}

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			Render500(w, r, "Failed to validate categories: "+err.Error())
			return
		}
		defer rows.Close()

		existingCategoryIDSet := make(map[int]bool)
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				log.Printf("CreatePostHandler: Error scanning category ID: %v", err)
				continue
			}
			existingCategoryIDSet[id] = true
		}

		// Check if all requested categories exist
		for id := range categoryIDSet {
			if !existingCategoryIDSet[id] {
				log.Printf("CreatePostHandler: Category %d does not exist", id)
				Render400(w, r, "One or more selected categories do not exist.")
				return
			}
		}

		tx, err := database.DB.Begin()
		if err != nil {
			Render500(w, r, "Failed to start transaction: "+err.Error())
			return
		}
		defer tx.Rollback()

		// Save trimmed content to avoid leading/trailing whitespace
		res, err := tx.Exec("INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)", user.ID, title, contentTrim)
		if err != nil {
			Render500(w, r, "Failed to create post: "+err.Error())
			return
		}
		postID, err := res.LastInsertId()
		if err != nil {
			Render500(w, r, "Failed to get post ID: "+err.Error())
			return
		}
		// No image handling

		for _, catID := range uniqueCategoryIDs {
			_, err = tx.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)", postID, catID)
			if err != nil {
				Render500(w, r, "Failed to link post to category: "+err.Error())
				return
			}
		}

		if err := tx.Commit(); err != nil {
			Render500(w, r, "Failed to commit transaction: "+err.Error())
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Если не POST и не GET, возвращаем 405
	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}

	rows, err := database.DB.Query("SELECT id, name FROM categories ORDER BY id")
	if err != nil {
		Render500(w, r, "Failed to load categories: "+err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Printf("Error scanning category for form: %v", err)
			continue
		}
		data.Categories = append(data.Categories, c)
	}
	renderTemplate(w, r, "create_post.html", data)
}

// PostDetailHandler (остается без изменений)
func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		Render400(w, r, "Invalid post ID format.")
		return
	}

	if r.Method == http.MethodPost {
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		commentContent := strings.TrimSpace(r.FormValue("comment"))
		commentRuneCount := countRunes(commentContent)
		if commentRuneCount == 0 || commentRuneCount > maxCommentLen {
			Render400(w, r, fmt.Sprintf("Comment must be between 1 and %d characters.", maxCommentLen))
			return
		}
		_, err := database.DB.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, user.ID, commentContent)
		if err != nil {
			Render500(w, r, "Failed to add comment: "+err.Error())
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}

	var post models.Post
	err = database.DB.QueryRow(`
        SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username,
               (SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND is_like = 1),
               (SELECT COUNT(*) FROM post_reactions WHERE post_id = p.id AND is_like = 0)
        FROM posts p JOIN users u ON p.user_id = u.id WHERE p.id = ?`, postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt, &post.Author, &post.Likes, &post.Dislikes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Render400(w, r, "Post not found.")
			return
		}
		Render500(w, r, "Failed to load post: "+err.Error())
		return
	}

	// Агрессивная обрезка: удаляем все пробелы и пустые строки в начале
	post.Content = strings.TrimSpace(post.Content)
	// Удаляем все пустые строки в начале
	lines := strings.Split(post.Content, "\n")
	var cleanedLines []string
	startFound := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Пропускаем пустые строки только в начале
		if !startFound && trimmedLine == "" {
			continue
		}
		startFound = true
		cleanedLines = append(cleanedLines, trimmedLine)
	}
	post.Content = strings.Join(cleanedLines, "\n")
	post.Content = strings.TrimSpace(post.Content)

	catRows, err := database.DB.Query(`SELECT c.id, c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?`, postID)
	if err == nil {
		for catRows.Next() {
			var cat models.Category
			if err := catRows.Scan(&cat.ID, &cat.Name); err == nil {
				post.Categories = append(post.Categories, cat)
			}
		}
		catRows.Close()
	}

	// Загружаем все комментарии с учетом удаленных
	commentRows, err := database.DB.Query(`
		SELECT co.id, co.post_id, co.user_id, co.content, co.created_at, u.username,
			   (SELECT COUNT(*) FROM comment_reactions WHERE comment_id = co.id AND is_like = 1),
			   (SELECT COUNT(*) FROM comment_reactions WHERE comment_id = co.id AND is_like = 0)
		FROM comments co JOIN users u ON co.user_id = u.id WHERE co.post_id = ? ORDER BY co.created_at ASC`, postID)
	var comments []models.Comment
	if err == nil {
		for commentRows.Next() {
			var c models.Comment
			if err := commentRows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.Author, &c.Likes, &c.Dislikes); err == nil {
				comments = append(comments, c)
			}
		}
		commentRows.Close()
	}

	data := TemplateData{User: user, Post: post, Comments: comments}
	renderTemplate(w, r, "post_detail.html", data)
}

// EditPostHandler (остается без изменений)
func EditPostHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		Render400(w, r, "Invalid post ID.")
		return
	}

	var post models.Post
	err = database.DB.QueryRow("SELECT id, user_id, title, content FROM posts WHERE id = ?", postID).Scan(&post.ID, &post.UserID, &post.Title, &post.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Render404(w, r)
			return
		}
		Render500(w, r, "Failed to load post: "+err.Error())
		return
	}

	if user.ID != post.UserID {
		Render403(w, r, "You are not authorized to edit this post.")
		return
	}

	// Проверяем метод PUT (method override уже обработан middleware)
	if r.Method == http.MethodPut {
		_ = r.ParseForm()

		newTitle := strings.TrimSpace(r.FormValue("title"))
		newContentRaw := r.FormValue("content")
		newContentNorm := strings.ReplaceAll(newContentRaw, "\r\n", "\n")
		// Удаляем все пробелы, табы и переносы строк в начале и конце
		newContentTrim := strings.TrimSpace(newContentNorm)
		// Дополнительно удаляем все пробелы и переносы строк в начале каждой строки
		lines := strings.Split(newContentTrim, "\n")
		var cleanedLines []string
		for i, line := range lines {
			trimmedLine := strings.TrimLeft(line, " \t")
			// Пропускаем пустые строки только в начале
			if i == 0 && trimmedLine == "" {
				continue
			}
			cleanedLines = append(cleanedLines, trimmedLine)
		}
		newContentTrim = strings.Join(cleanedLines, "\n")
		// Финальная обрезка
		newContentTrim = strings.TrimSpace(newContentTrim)
		titleRuneCount := countRunes(newTitle)
		contentRuneCount := countRunes(newContentTrim)
		log.Printf("EditPostHandler: user %d, postID %d, newTitle runes=%d, newContent runes=%d", user.ID, postID, titleRuneCount, contentRuneCount)
		if titleRuneCount == 0 || titleRuneCount > maxTitleLen {
			Render400(w, r, fmt.Sprintf("Title must be between 1 and %d characters.", maxTitleLen))
			return
		}
		if contentRuneCount == 0 || contentRuneCount > maxContentLen {
			Render400(w, r, fmt.Sprintf("Content must be between 1 and %d characters.", maxContentLen))
			return
		}
		// Save trimmed content to avoid leading/trailing whitespace
		_, err := database.DB.Exec("UPDATE posts SET title = ?, content = ? WHERE id = ?", newTitle, newContentTrim, postID)
		if err != nil {
			Render500(w, r, "Failed to update post: "+err.Error())
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
		return
	}

	// Если не PUT и не GET, возвращаем 405
	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}

	log.Printf("EditPostHandler: user=%d, postID=%d, post.Title='%s', post.Content len=%d", user.ID, post.ID, post.Title, len(post.Content))
	renderTemplate(w, r, "edit_post.html", TemplateData{User: user, Post: post})
}

// DeletePostHandler (остается без изменений)
func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод DELETE (method override уже обработан middleware)
	if r.Method != http.MethodDelete {
		Render405(w, r)
		return
	}

	user := auth.GetUserFromContext(r.Context())
	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		Render400(w, r, "Invalid post ID.")
		return
	}

	var postUserID int
	err = database.DB.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&postUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Render404(w, r)
			return
		}
		Render500(w, r, "Failed to verify ownership: "+err.Error())
		return
	}
	if user.ID != postUserID {
		Render403(w, r, "You are not authorized to delete this post.")
		return
	}

	_, err = database.DB.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		Render500(w, r, "Failed to delete post: "+err.Error())
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ReactionHandler returns JSON response instead of redirecting
func ReactionHandler(w http.ResponseWriter, r *http.Request, isPost bool) {
	// Проверяем метод GET (лайки используют GET через JavaScript fetch)
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		log.Printf("ReactionHandler: Unauthorized request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Unauthorized",
		})
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		log.Printf("ReactionHandler: Invalid ID: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid ID",
		})
		return
	}
	isLike := r.URL.Query().Get("like") == "1"
	log.Printf("ReactionHandler: User %d, type=%v, id=%d, isLike=%v", user.ID, isPost, id, isLike)

	var tableName, colName string
	if isPost {
		tableName, colName = "post_reactions", "post_id"
	} else {
		tableName, colName = "comment_reactions", "comment_id"
	}

	tx, err := database.DB.Begin()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Check for existing reaction within transaction
	var existingIsLike sql.NullBool
	query := fmt.Sprintf("SELECT is_like FROM %s WHERE %s = ? AND user_id = ?", tableName, colName)
	err = tx.QueryRow(query, id, user.ID).Scan(&existingIsLike)

	if err == nil && existingIsLike.Valid {
		// Reaction exists
		if existingIsLike.Bool == isLike {
			// Same reaction - remove it
			query = fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND user_id = ?", tableName, colName)
			_, err = tx.Exec(query, id, user.ID)
		} else {
			// Different reaction - update it
			query = fmt.Sprintf("UPDATE %s SET is_like = ?, dislike = ? WHERE %s = ? AND user_id = ?", tableName, colName)
			dislike := !isLike
			_, err = tx.Exec(query, isLike, dislike, id, user.ID)
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		// No reaction exists - insert new one
		query = fmt.Sprintf("INSERT INTO %s (%s, user_id, is_like, dislike) VALUES (?, ?, ?, ?)", tableName, colName)
		dislike := !isLike
		_, err = tx.Exec(query, id, user.ID, isLike, dislike)
	} else {
		// Database error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Database error checking reaction",
		})
		return
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to update reaction",
		})
		return
	}
	if err = tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to commit reaction",
		})
		return
	}

	// Get updated counts
	var likes, dislikes int
	countQuery := fmt.Sprintf(`
		SELECT 
			(SELECT COUNT(*) FROM %s WHERE %s = ? AND is_like = 1),
			(SELECT COUNT(*) FROM %s WHERE %s = ? AND is_like = 0)
	`, tableName, colName, tableName, colName)
	err = database.DB.QueryRow(countQuery, id, id).Scan(&likes, &dislikes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to get updated counts",
		})
		return
	}

	// Return JSON response
	log.Printf("ReactionHandler: Success - likes=%d, dislikes=%d", likes, dislikes)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"likes":    likes,
		"dislikes": dislikes,
	})
}

func LikePostHandler(w http.ResponseWriter, r *http.Request)    { ReactionHandler(w, r, true) }
func LikeCommentHandler(w http.ResponseWriter, r *http.Request) { ReactionHandler(w, r, false) }

// EditCommentHandler (остается без изменений)
func EditCommentHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	commentID, err1 := strconv.Atoi(r.URL.Query().Get("id"))
	postID, err2 := strconv.Atoi(r.URL.Query().Get("post"))
	if err1 != nil || err2 != nil {
		Render400(w, r, "Invalid IDs.")
		return
	}

	var c models.Comment
	err := database.DB.QueryRow("SELECT id, user_id, content FROM comments WHERE id = ?", commentID).Scan(&c.ID, &c.UserID, &c.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Render404(w, r)
			return
		}
		Render500(w, r, "Failed to load comment: "+err.Error())
		return
	}

	if user.ID != c.UserID {
		Render403(w, r, "You are not authorized to edit this comment.")
		return
	}
	c.PostID = postID

	// Проверяем метод PUT (method override уже обработан middleware)
	if r.Method == http.MethodPut {
		if err := r.ParseForm(); err != nil {
			Render500(w, r, "Failed to parse form: "+err.Error())
			return
		}
		newContent := strings.TrimSpace(r.FormValue("content"))
		contentRuneCount := countRunes(newContent)
		log.Printf("EditCommentHandler: user %d, commentID %d, postID %d, newContent runes=%d", user.ID, commentID, postID, contentRuneCount)
		if contentRuneCount == 0 || contentRuneCount > maxCommentLen {
			Render400(w, r, fmt.Sprintf("Comment must be between 1 and %d characters.", maxCommentLen))
			return
		}
		_, err := database.DB.Exec("UPDATE comments SET content = ? WHERE id = ?", newContent, commentID)
		if err != nil {
			Render500(w, r, "Failed to update comment: "+err.Error())
			return
		}
		log.Printf("EditCommentHandler: Successfully updated comment %d", commentID)
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
		return
	}

	// Если не PUT и не GET, возвращаем 405
	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}

	log.Printf("EditCommentHandler GET: user=%d, commentID=%d, postID=%d, content length=%d, content preview=%.50s", user.ID, c.ID, postID, len(c.Content), c.Content)
	if c.ID == 0 {
		Render500(w, r, "Comment data is invalid")
		return
	}
	renderTemplate(w, r, "edit_comment.html", TemplateData{User: user, Comment: c, Post: models.Post{ID: postID}})
}

// DeleteCommentHandler выполняет мягкое удаление комментария
func DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод DELETE (method override уже обработан middleware)
	if r.Method != http.MethodDelete {
		Render405(w, r)
		return
	}

	user := auth.GetUserFromContext(r.Context())
	commentID, err1 := strconv.Atoi(r.URL.Query().Get("id"))
	postID, err2 := strconv.Atoi(r.URL.Query().Get("post"))
	if err1 != nil || err2 != nil {
		Render400(w, r, "Invalid IDs.")
		return
	}

	var commentUserID int
	err := database.DB.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&commentUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Render404(w, r)
			return
		}
		Render500(w, r, "Failed to verify ownership: "+err.Error())
		return
	}
	if user.ID != commentUserID {
		Render403(w, r, "You are not authorized to delete this comment.")
		return
	}

	_, err = database.DB.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		Render500(w, r, "Failed to delete comment: "+err.Error())
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

// ProtectStatic - защищает статические файлы и устанавливает правильные MIME типы
func ProtectStatic(fs http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			Render404(w, r)
			return
		}
		if !strings.HasSuffix(r.URL.Path, ".css") && !strings.HasSuffix(r.URL.Path, ".ico") && !strings.HasSuffix(r.URL.Path, ".js") {
			Render404(w, r)
			return
		}

		// Устанавливаем правильные MIME типы
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(r.URL.Path, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}

		fs.ServeHTTP(w, r)
	}
}
