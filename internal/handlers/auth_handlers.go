package handlers

import (
	"errors"
	"log"
	"net/http"

	"forum/internal/auth"
)

// RegisterHandler displays and processes the registration form.
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, r, "register.html", TemplateData{})
		return
	}
	if r.Method != http.MethodPost {
		Render405(w, r)
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	_, err := auth.RegisterUser(email, username, password)
	if err != nil {
		log.Printf("Registration error: %v", err)
		var errMsg string
		if errors.Is(err, auth.ErrEmailExists) {
			errMsg = "Email already registered."
		} else if errors.Is(err, auth.ErrUsernameExists) {
			errMsg = "Username already taken."
		} else if errors.Is(err, auth.ErrInvalidInput) {
			errMsg = err.Error()
		} else {
			errMsg = "Registration failed due to a server error."
		}
		w.WriteHeader(http.StatusBadRequest)
		renderTemplate(w, r, "register.html", TemplateData{Error: errMsg})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LoginHandler displays and processes the login form.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, r, "login.html", TemplateData{})
		return
	}
	if r.Method != http.MethodPost {
		Render405(w, r)
		return
	}

	login := r.FormValue("login") // Can be email or username
	password := r.FormValue("password")

	user, session, err := auth.LoginUser(login, password)
	if err != nil {
		log.Printf("Login error for %s: %v", login, err)
		errMsg := "Invalid email/username or password."
		w.WriteHeader(http.StatusUnauthorized)
		renderTemplate(w, r, "login.html", TemplateData{Error: errMsg})
		return
	}

	// Set the session cookie
	auth.SetSessionCookie(w, session.UUID, session.Expires)
	log.Printf("User '%s' (ID: %d) logged in successfully.", user.Username, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler logs out the user by deleting their session.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Render405(w, r)
		return
	}
	sessionCookie, err := r.Cookie("session_token")
	if err == nil { // If cookie exists, try to delete session from DB
		err = auth.LogoutUser(sessionCookie.Value)
		if err != nil && !errors.Is(err, auth.ErrSessionNotFound) {
			log.Printf("Error deleting session from DB: %v", err)
		}
	}

	// Always clear the cookie from the client
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
