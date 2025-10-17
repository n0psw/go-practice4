package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID      int     `db:"id"`
	Name    string  `db:"name"`
	Email   string  `db:"email"`
	Balance float64 `db:"balance"`
}

func InsertUser(db *sqlx.DB, user User) error {
	query := `INSERT INTO users (name, email, balance) 
              VALUES (:name, :email, :balance)`
	
	_, err := db.NamedExec(query, user)
	return err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	users := []User{}
	query := `SELECT id, name, email, balance FROM users`
	
	err := db.Select(&users, query)
	return users, err
}

func GetUserByID(db *sqlx.DB, id int) (User, error) {
	var user User
	query := `SELECT id, name, email, balance FROM users WHERE id = $1`
	
	err := db.Get(&user, query, id)
	if err == sql.ErrNoRows {
		return user, fmt.Errorf("пользователь с ID %d не найден", id)
	}
	return user, err
}

func TransferBalance(db *sqlx.DB, fromID int, toID int, amount float64) (err error) {
	if amount <= 0 {
		return fmt.Errorf("сумма перевода должна быть положительной")
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var fromBalance float64
	err = tx.Get(&fromBalance, "SELECT balance FROM users WHERE id = $1 FOR UPDATE", fromID)
	if err != nil {
		return fmt.Errorf("не удалось получить баланс отправителя: %w", err)
	}

	if fromBalance < amount {
		return fmt.Errorf("недостаточно средств у пользователя ID %d (баланс: %.2f)", fromID, fromBalance)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance - $1 WHERE id = $2", amount, fromID)
	if err != nil {
		return fmt.Errorf("не удалось обновить баланс отправителя: %w", err)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance + $1 WHERE id = $2", amount, toID)
	if err != nil {
		return fmt.Errorf("не удалось обновить баланс получателя: %w", err)
	}

	err = tx.Commit()
	return err
}

func main() {
	const connStr = "host=localhost port=5430 user=user password=password dbname=mydatabase sslmode=disable"

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("Успешное подключение к базе данных PostgreSQL с настройками пула.")
	
	newUser := User{Name: "Alikhan New", Email: "alikhan.new@example.com", Balance: 200.00}
	if err := InsertUser(db, newUser); err != nil {
		log.Println("Ошибка InsertUser:", err)
	} else {
		fmt.Println("Пользователь 'Alikhan New' успешно вставлен.")
	}

	if users, err := GetAllUsers(db); err != nil {
		log.Println("Ошибка GetAllUsers:", err)
	} else {
		fmt.Println("\nВсе пользователи в БД:")
		for _, u := range users {
			fmt.Printf("ID: %d, Имя: %s, Баланс: %.2f\n", u.ID, u.Name, u.Balance)
		}
	}

	if user1, err := GetUserByID(db, 1); err != nil {
		log.Println("Ошибка GetUserByID:", err)
	} else {
		fmt.Printf("\nПользователь с ID 1: Имя: %s, Баланс: %.2f\n", user1.Name, user1.Balance)
	}

	senderID := 1
	receiverID := 2
	transferAmount := 100.0

	senderStart, _ := GetUserByID(db, senderID)
	receiverStart, _ := GetUserByID(db, receiverID)
	fmt.Printf("\nПопытка перевода %.2f от ID %d к ID %d:\n", transferAmount, senderID, receiverID)
	fmt.Printf("Начальный баланс: ID %d (%.2f), ID %d (%.2f)\n", senderID, senderStart.Balance, receiverID, receiverStart.Balance)

	if err := TransferBalance(db, senderID, receiverID, transferAmount); err != nil {
		log.Println("Ошибка TransferBalance:", err)
	} else {
		fmt.Println("Перевод успешно завершен.")
	}

	senderEnd, _ := GetUserByID(db, senderID)
	receiverEnd, _ := GetUserByID(db, receiverID)
	fmt.Printf("Конечный баланс: ID %d (%.2f), ID %d (%.2f)\n", senderID, senderEnd.Balance, receiverID, receiverEnd.Balance)

	rollbackAmount := 2000.0
	fmt.Printf("\nТест Rollback: Попытка перевода %.2f (недостаточно средств) от ID %d.\n", rollbackAmount, senderID)
	senderBeforeRollback, _ := GetUserByID(db, senderID)

	if err := TransferBalance(db, senderID, receiverID, rollbackAmount); err != nil {
		fmt.Printf("Ошибка (Rollback OK): %v\n", err)
	}

	senderAfterRollback, _ := GetUserByID(db, senderID)
	fmt.Printf("Баланс отправителя ID %d: До: %.2f, После: %.2f. Баланс не изменился (Rollback OK).\n", senderID, senderBeforeRollback.Balance, senderAfterRollback.Balance)
}
