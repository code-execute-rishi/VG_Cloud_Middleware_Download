package queries

import (
	"backend/models"
	"database/sql"
	"errors"
	"time"
)

func GetOrCreateUser(db *sql.DB, clerkUserID, email string) (string, error) {
	var userID string

	err := db.QueryRow("SELECT id FROM users WHERE clerk_user_id = $1", clerkUserID).Scan(&userID)

	if err == sql.ErrNoRows {
		err = db.QueryRow(
			"INSERT INTO users (clerk_user_id, email) VALUES ($1, $2) RETURNING id",
			clerkUserID, email,
		).Scan(&userID)
		return userID, err
	}

	return userID, err
}

func ClaimDevice(db *sql.DB, pairingCode int, ownerID, name string) (string, error) {
	var deviceID string

	err := db.QueryRow(
		"UPDATE devices SET owner_id = $1, name = $2 WHERE pairing_code = $3 AND owner_id IS NULL RETURNING id",
		ownerID, name, pairingCode,
	).Scan(&deviceID)

	if err == sql.ErrNoRows {
		return "", errors.New("device not found or already claimed")
	}

	return deviceID, err
}

func GetUserDevices(db *sql.DB, userID string) ([]models.Device, error) {
	query := `
        SELECT 
            d.id, d.name, d.status, d.last_seen,
            dt.latitude, dt.longitude, dt.altitude, dt.speed, 
            dt.heading, dt.signal_strength, dt.battery, dt.armed, dt.flight_mode
        FROM devices d
        LEFT JOIN device_telemetry dt ON d.id = dt.device_id
        WHERE d.owner_id = $1
        ORDER BY d.created_at DESC
    `

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		var lat, lng, alt, spd, hdg sql.NullFloat64
		var sig, bat sql.NullInt64
		var armed sql.NullBool
		var flightMode sql.NullString

		err := rows.Scan(
			&d.ID, &d.Name, &d.Status, &d.LastSeen,
			&lat, &lng, &alt, &spd, &hdg, &sig, &bat,
			&armed, &flightMode,
		)
		if err != nil {
			return nil, err
		}

		// Convert nullable values to pointers
		if lat.Valid {
			d.Latitude = &lat.Float64
		}
		if lng.Valid {
			d.Longitude = &lng.Float64
		}
		if alt.Valid {
			d.Altitude = &alt.Float64
		}
		if spd.Valid {
			d.Speed = &spd.Float64
		}
		if hdg.Valid {
			d.Heading = &hdg.Float64
		}
		if sig.Valid {
			sigInt := int(sig.Int64)
			d.SignalStrength = &sigInt
		}
		if bat.Valid {
			batInt := int(bat.Int64)
			d.Battery = &batInt
		}
		if armed.Valid {
			d.Armed = &armed.Bool
		}
		if flightMode.Valid {
			d.FlightMode = &flightMode.String
		}

		// Liveness Check: If last seen > 1 minute ago, force status to "Offline"
		if d.LastSeen != nil {
			if time.Since(*d.LastSeen) > 1*time.Minute {
				offline := "Offline"
				d.Status = offline // Use Status field or FlightMode field?
				// The frontend uses flight_mode override if present.
				// So we should set flight_mode to "Offline" too if we want to be sure.
				d.FlightMode = &offline

				// Also disarm
				disarmed := false
				d.Armed = &disarmed
			}
		}

		devices = append(devices, d)
	}

	return devices, nil
}

func GetDeviceByID(db *sql.DB, deviceID, userID string) (*models.DeviceDetail, error) {
	query := `
        SELECT d.id, d.name, d.status, d.last_seen,
               dt.latitude, dt.longitude, dt.battery
        FROM devices d
        LEFT JOIN device_telemetry dt ON d.id = dt.device_id
        WHERE d.id = $1 AND d.owner_id = $2
    `

	var device models.DeviceDetail
	var lat, lng sql.NullFloat64
	var battery sql.NullInt64

	err := db.QueryRow(query, deviceID, userID).Scan(
		&device.ID, &device.Name, &device.Status, &device.LastSeen,
		&lat, &lng, &battery,
	)

	if lat.Valid {
		device.Latitude = &lat.Float64
	}
	if lng.Valid {
		device.Longitude = &lng.Float64
	}
	if battery.Valid {
		batteryInt := int(battery.Int64)
		device.Battery = &batteryInt
	}

	return &device, err
}

func DeleteDevice(db *sql.DB, deviceID, userID string) error {
	result, err := db.Exec(
		"UPDATE devices SET owner_id = NULL, name = NULL WHERE id = $1 AND owner_id = $2",
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

func GetOrCreateUserByEmail(db *sql.DB, email string) (string, error) {

	var userID string
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err == sql.ErrNoRows {
		err = db.QueryRow(
			"INSERT INTO users (email) VALUES ($1) RETURNING id",
			email,
		).Scan(&userID)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
	}
	return userID, nil
}
