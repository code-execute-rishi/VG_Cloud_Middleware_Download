package queries

import (
	"backend/utils"
	"database/sql"
	"errors"
	"time"
)

func DeviceExistsByID(db *sql.DB, deviceID string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1)", deviceID).Scan(&exists)
	return exists, err
}

func DeviceExistsByPublicKey(db *sql.DB, publicKey string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM devices WHERE public_key = $1)", publicKey).Scan(&exists)
	return exists, err
}

func PairingCodeExists(db *sql.DB, pairingCode int) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM devices WHERE pairing_code = $1)", pairingCode).Scan(&exists)
	return exists, err
}

func CreateDevice(db *sql.DB, publicKey string, pairingCode int, nodeID string) (string, error) {
	var deviceID string
	err := db.QueryRow("INSERT INTO devices (public_key, pairing_code, node_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
		publicKey, pairingCode, nodeID, time.Now()).Scan(&deviceID)
	return deviceID, err
}

func GenerateChallenge(db *sql.DB, deviceID string) (string, error) {
	challenge := utils.GenerateChallenge()
	_, err := db.Exec("INSERT INTO challenges (device_id, challenge) VALUES ($1, $2) ON CONFLICT (device_id) DO UPDATE SET challenge = EXCLUDED.challenge",
		deviceID, challenge)
	return challenge, err
}

func GetChallenge(db *sql.DB, deviceID string) (string, error) {
	var challenge string
	var expiresAt time.Time
	err := db.QueryRow("SELECT challenge, expires_at FROM challenges WHERE device_id = $1", deviceID).Scan(&challenge, &expiresAt)
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		return "", errors.New("challenge expired")
	}
	return challenge, nil
}

func GetPublicKey(db *sql.DB, deviceID string) (string, error) {
	var publicKey string
	err := db.QueryRow("SELECT public_key FROM devices WHERE id = $1", deviceID).Scan(&publicKey)
	return publicKey, err
}

func DeleteChallenge(db *sql.DB, deviceID string) error {
	_, err := db.Exec("DELETE FROM challenges WHERE device_id = $1", deviceID)
	return err
}

func DeleteDeviceHard(db *sql.DB, deviceID, userID string) error {
	result, err := db.Exec(
		"DELETE FROM devices WHERE id = $1 AND owner_id = $2",
		deviceID, userID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("device not found or not owned by user")
	}

	return nil
}

func GetUserIDFromDeviceID(db *sql.DB, deviceID string) (string, error) {
	var userID sql.NullString

	err := db.QueryRow(
		"SELECT owner_id FROM devices WHERE id = $1",
		deviceID,
	).Scan(&userID)

	if err != nil {
		return "", err
	}

	if !userID.Valid {
		return "", errors.New("device is not yet claimed by any user")
	}

	return userID.String, nil
}

func CheckClaim(db *sql.DB, deviceID string) (bool, error) {
	var ownerID sql.NullString

	query := `SELECT owner_id FROM devices WHERE id = $1`
	err := db.QueryRow(query, deviceID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	if ownerID.Valid && ownerID.String != "" {
		return true, nil
	}

	return false, nil
}

func UpdateDeviceStatus(db *sql.DB, deviceID, status string) error {
	query := `
        UPDATE devices 
        SET status = $1, last_seen = NOW()
        WHERE id = $2
    `
	_, err := db.Exec(query, status, deviceID)
	return err
}
