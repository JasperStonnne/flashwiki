package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/db"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	minPasswordLength = 8
	minNameLength     = 1
	maxNameLength     = 50
)

func main() {
	var email string
	var password string
	var name string

	flag.StringVar(&email, "email", "", "manager email")
	flag.StringVar(&password, "password", "", "manager password")
	flag.StringVar(&name, "name", "", "manager display name")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: go run ./cmd/seed --email=<email> --password=<password> --name=<display_name>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	name = strings.TrimSpace(name)

	if err := validateInput(email, password, name); err != nil {
		flag.Usage()
		log.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepo(pool)

	existing, err := userRepo.FindByEmail(ctx, email)
	if err != nil {
		log.Fatal(err)
	}
	if existing != nil {
		fmt.Printf("user %s already exists, skipping\n", email)
		os.Exit(0)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal(err)
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		DisplayName:  name,
		Role:         "manager",
		Locale:       "zh",
	}

	if err := userRepo.CreateUser(ctx, &user); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("manager user created: %s (%s)\n", email, user.ID)
}

func validateInput(email, password, name string) error {
	if email == "" {
		return fmt.Errorf("--email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("--email is invalid")
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		return fmt.Errorf("--password must be at least %d characters", minPasswordLength)
	}

	nameLen := utf8.RuneCountInString(name)
	if nameLen < minNameLength || nameLen > maxNameLength {
		return fmt.Errorf("--name must be between %d and %d characters", minNameLength, maxNameLength)
	}

	return nil
}
