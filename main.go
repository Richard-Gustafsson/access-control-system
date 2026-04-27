package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Richard-Gustafsson/access-control-system/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

func main() {
	dbURL := "postgres://user:password@localhost:5432/access-control-system?sslmode=disable"

	// 1. Run migrations
	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		log.Fatalf("Could not create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Could not run migration up: %v", err)
	}
	fmt.Println("Migrations completed!")

	// 2. Connect to test (just like before)
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println("Database is ready and tables are in place!")

	// 1. Create a new user
	newUser, err := CreateUser(context.Background(), conn, "Bob Berglund", "staff")
	if err != nil {
		log.Fatalf("Could not create user: %v", err)
	}
	fmt.Printf("Created: %s\n", newUser.Name)

	// 2. Fetch the user we just created with the same ID
	fetchedUser, err := GetUser(context.Background(), conn, newUser.ID)
	if err != nil {
		log.Fatalf("Could not fetch user: %v", err)
	}
	fmt.Printf("Fetched from DB: %s (Role: %s)\n", fetchedUser.Name, fetchedUser.Role)

	// Skapa en dörr till serverrummet
	newDoor, err := CreateDoor(context.Background(), conn, "Serverrum Söder", "Källare vån -1", "admin")
	if err != nil {
		log.Fatalf("Could not create door: %v", err)
	}

	fmt.Printf("Door created! ID: %s, Name: %s (Requires role: %s)\n",
		newDoor.ID, newDoor.Name, newDoor.MinRoleRequired)

	// Testa att låta Bob försöka gå in i Serverrummet
	// (Använd ID:n från de variabler du skapade tidigare: newUser.ID och newDoor.ID)
	allowed, err := AttemptAccess(context.Background(), conn, newUser.ID, newDoor.ID)
	if err != nil {
		log.Printf("Systemfel vid inpassering: %v", err)
	}

	if allowed {
		fmt.Println("Välkommen in! Dörren är upplåst.")
	} else {
		fmt.Println("Åtkomst nekas! Detta försök har loggats.")
	}
}

// CreateUser takes in a database connection and user details, creates a new user in the database, and returns the created user with its ID and creation timestamp.
func CreateUser(ctx context.Context, conn *pgx.Conn, name string, role string) (*models.User, error) {
	user := &models.User{
		Name: name,
		Role: role,
	}

	// SQL query that saves the user and returns the auto-generated ID and timestamp
	query := `
		INSERT INTO users (name, role) 
		VALUES ($1, $2) 
		RETURNING id, created_at`

	err := conn.QueryRow(ctx, query, user.Name, user.Role).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("could not create user: %v", err)
	}

	return user, nil
}

func CreateDoor(ctx context.Context, conn *pgx.Conn, name, location, minRole string) (*models.Door, error) {
	door := &models.Door{
		Name:            name,
		Location:        location,
		MinRoleRequired: minRole,
	}

	query := `
		INSERT INTO doors (name, location, min_role_required)
		VALUES ($1, $2, $3)
		RETURNING id`

	// We send the address of door.ID so the database can fill it in after the insert
	err := conn.QueryRow(ctx, query, door.Name, door.Location, door.MinRoleRequired).Scan(&door.ID)
	if err != nil {
		return nil, fmt.Errorf("could not create door: %v", err)
	}

	return door, nil
}

// GetUser fetches a user from the database based on their UUID
func GetUser(ctx context.Context, conn *pgx.Conn, id string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, name, role, created_at 
		FROM users 
		WHERE id = $1`

	err := conn.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no user found with ID: %s", id)
		}
		return nil, fmt.Errorf("error fetching user: %v", err)
	}

	return user, nil
}

// GetDoor fetches a door from the database based on its UUID
func GetDoor(ctx context.Context, conn *pgx.Conn, id string) (*models.Door, error) {
	door := &models.Door{}

	query := `
		SELECT id, name, location, min_role_required 
		FROM doors 
		WHERE id = $1`

	err := conn.QueryRow(ctx, query, id).Scan(&door.ID, &door.Name, &door.Location, &door.MinRoleRequired)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no door found with ID: %s", id)
		}
		return nil, fmt.Errorf("error fetching door: %v", err)
	}

	return door, nil
}

func AttemptAccess(ctx context.Context, conn *pgx.Conn, userID string, doorID string) (bool, error) {
	// 1.Fetch user and door details
	user, err := GetUser(ctx, conn, userID)
	if err != nil {
		return false, fmt.Errorf("Could not verify user: %v", err)
	}

	door, err := GetDoor(ctx, conn, doorID)
	if err != nil {
		return false, fmt.Errorf("Could not verify door: %v", err)
	}

	// 2. Check access logic based on user role and door requirements
	// Simple logic: If the door requires 'admin', the user must be 'admin'.
	// If the door requires 'staff', both 'staff' and 'admin' can enter.
	granted := false
	if door.MinRoleRequired == "admin" && user.Role == "admin" {
		granted = true
	} else if door.MinRoleRequired == "staff" && (user.Role == "staff" || user.Role == "admin") {
		granted = true
	}

	// 3. Save the access attempt in the access_logs table
	query := `
		INSERT INTO access_logs (user_id, door_id, granted)
		VALUES ($1, $2, $3)`

	_, err = conn.Exec(ctx, query, userID, doorID, granted)
	if err != nil {
		return false, fmt.Errorf("Could not log access attempt: %v", err)
	}

	return granted, nil
}
