package queries

import (
    "backend/models"
    "database/sql"
)

func SaveZeroTierConfig(db *sql.DB, deviceID, zerotierIP string) error {
    query := `
        INSERT INTO zerotier_config (device_id, zerotier_ip)
        VALUES ($1, $2)
        ON CONFLICT (device_id)
        DO UPDATE SET 
            zerotier_ip = EXCLUDED.zerotier_ip,
            updated_at = NOW()
    `
    _, err := db.Exec(query, deviceID, zerotierIP)
    return err
}

func GetZeroTierConfig(db *sql.DB, deviceID string) (*models.ZerotierConfig, error) {
    var config models.ZerotierConfig
    
    query := `
        SELECT zerotier_ip
        FROM zerotier_config
        WHERE device_id = $1
    `
    
    err := db.QueryRow(query, deviceID).Scan(&config.ZerotierIP)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    
    return &config, nil
}

func GetDeviceNodeID(db *sql.DB, deviceID string) (string, error) {
    var nodeID string
    err := db.QueryRow("SELECT node_id FROM devices WHERE id = $1", deviceID).Scan(&nodeID)
    return nodeID, err
}