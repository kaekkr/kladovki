// Command superadmin creates a platform operator (role superadmin), or sets a
// new password for an existing one, directly in the database. Run it once
// from your own machine — no password is stored in env vars or in the repo:
//
//	cd backend
//	DB_PATH='postgres://…' go run ./cmd/superadmin --email you@example.com
//
// The password is asked twice without echo. Only DB_PATH is read (the
// server's other settings, e.g. JWT_SECRET, are not needed).
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/database"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

func main() {
	email := flag.String("email", "", "operator's login email (required)")
	flag.Parse()
	if err := run(strings.TrimSpace(*email)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(email string) error {
	if email == "" {
		return errors.New("usage: go run ./cmd/superadmin --email you@example.com")
	}
	dsn := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dsn == "" {
		return errors.New("DB_PATH is not set")
	}

	password, err := askPassword()
	if err != nil {
		return err
	}

	db, err := database.Open(dsn) // also applies migrations (needs the superadmin role)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := service.New(repository.New(db)).SetSuperadmin(ctx, email, password)
	if err != nil {
		return err
	}
	switch res {
	case service.SuperadminCreated:
		fmt.Printf("Created superadmin %s. Log in at <site>/login → /platform.\n", strings.ToLower(email))
	case service.SuperadminPasswordReset:
		fmt.Printf("Password changed for superadmin %s (other sessions are signed out).\n", email)
	case service.SuperadminConflict:
		return fmt.Errorf("%s belongs to an ОСИ or a resident; nothing changed", email)
	}
	return nil
}

// askPassword reads the password twice from the terminal without echo.
func askPassword() (string, error) {
	in := bufio.NewReader(os.Stdin)
	first, err := readHidden(in, "Password (min 10 chars): ")
	if err != nil {
		return "", err
	}
	second, err := readHidden(in, "Repeat password: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("passwords do not match")
	}
	return first, nil
}

func readHidden(in *bufio.Reader, prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	echo(false)
	line, err := in.ReadString('\n')
	echo(true)
	fmt.Fprintln(os.Stderr)
	if err != nil && line == "" {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// echo turns terminal echo on/off (macOS/Linux). When stdin is not a
// terminal (piped input) stty fails and the input is simply read.
func echo(on bool) {
	arg := "-echo"
	if on {
		arg = "echo"
	}
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}
