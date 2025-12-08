package queries

import (
    "backend/models"
    "database/sql"
    "errors"
)

func IsDeviceOwned(db *sql.DB, deviceID string) (bool, error) {
    var ownerID sql.NullString
    err := db.QueryRow("SELECT owner_id FROM devices WHERE id = $1", deviceID).Scan(&ownerID)
    if err != nil {
        if err == sql.ErrNoRows {
            return false, nil
        }
        return false, err
    }
    return ownerID.Valid, nil
}

func IsDeviceOwner(db *sql.DB, deviceID, userID string) (bool, error) {
    var ownerID string
    err := db.QueryRow("SELECT owner_id FROM devices WHERE id = $1", deviceID).Scan(&ownerID)
    if err != nil {
        return false, err
    }
    return ownerID == userID, nil
}

func GetCollaborators(db *sql.DB, deviceID string) ([]models.Collaborator, error) {
    query := `
        SELECT c.id, u.email, c.added_at
        FROM collaborators c
        JOIN users u ON c.user_id = u.id
        WHERE c.device_id = $1
        ORDER BY c.added_at DESC
    `
    
    rows, err := db.Query(query, deviceID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var collaborators []models.Collaborator
    for rows.Next() {
        var c models.Collaborator
        err := rows.Scan(&c.ID, &c.Email, &c.AddedAt)  // Added &c.ID
        if err != nil {
            return nil, err
        }
        collaborators = append(collaborators, c)
    }
    
    return collaborators, nil
}

func AddCollaborator(db *sql.DB, deviceID, email string) (string, error) {
    var userID string
    err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
    if err == sql.ErrNoRows {
        return "", errors.New("user with this email doesn't exist")
    }
    if err != nil {
        return "", err
    }
    
    var collaboratorID string
    err = db.QueryRow(
        "INSERT INTO collaborators (device_id, user_id, email) VALUES ($1, $2, $3) RETURNING id",
        deviceID, userID, email,
    ).Scan(&collaboratorID)
    
    if err != nil {
        return "", err
    }
    
    return collaboratorID, nil
}

func RemoveCollaborator(db *sql.DB, deviceID, email string) error {
    result, err := db.Exec(`
        DELETE FROM collaborators 
        WHERE device_id = $1 
        AND user_id = (SELECT id FROM users WHERE email = $2)
    `, deviceID, email)
    
    if err != nil {
        return err
    }
    
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return errors.New("collaborator not found")
    }
    
    return nil
}