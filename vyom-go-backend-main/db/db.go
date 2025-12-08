package db

import (
    "database/sql"
    "fmt"
    "log"
    
    _ "github.com/lib/pq"
)

var DB *sql.DB

func Connect(host, port, user, password, dbname string) error {
    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname,
    )
    
    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return err
    }
    
    err = DB.Ping()
    if err != nil {
        return err
    }
    
    log.Println("✅ Database connected!")
    return nil
}