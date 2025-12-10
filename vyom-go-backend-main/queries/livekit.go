package queries

import (
    "database/sql"
)

func UserHasDeviceAccess(db *sql.DB, deviceID, userID string) (bool, error) {
    var count int
    query := `
        SELECT COUNT(*) FROM (
            SELECT 1 FROM devices WHERE id = $1 AND owner_id = $2
            UNION
            SELECT 1 FROM collaborators WHERE device_id = $1 AND user_id = $2
        ) AS access
    `
    err := db.QueryRow(query, deviceID, userID).Scan(&count)
    return count > 0, err
}

func GetOrCreateLiveKitRoom(db *sql.DB, deviceID string) (string, error) {
    var roomName string
    
    err := db.QueryRow("SELECT room_name FROM livekit_rooms WHERE device_id = $1", deviceID).Scan(&roomName)
    
    if err == sql.ErrNoRows {
        roomName = deviceID
        _, err = db.Exec(
            "INSERT INTO livekit_rooms (device_id, room_name) VALUES ($1, $2)",
            deviceID, roomName,
        )
        return roomName, err
    }
    
    return roomName, err
}