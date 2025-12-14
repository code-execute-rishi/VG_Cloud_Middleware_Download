package queries

import (
	"backend/models"
	"database/sql"
	"fmt"
	"strings"
)

func UpdateDeviceTelemetry(db *sql.DB, update models.TelemetryUpdate) error {
	var setClauses []string
	var insertColumns []string
	var insertPlaceholders []string
	var args []interface{}
	argPosition := 1

	if update.Latitude != nil {
		insertColumns = append(insertColumns, "latitude")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("latitude = $%d", argPosition))
		args = append(args, *update.Latitude)
		argPosition++
	}

	if update.Longitude != nil {
		insertColumns = append(insertColumns, "longitude")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("longitude = $%d", argPosition))
		args = append(args, *update.Longitude)
		argPosition++
	}

	if update.Altitude != nil {
		insertColumns = append(insertColumns, "altitude")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("altitude = $%d", argPosition))
		args = append(args, *update.Altitude)
		argPosition++
	}

	if update.Speed != nil {
		insertColumns = append(insertColumns, "speed")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("speed = $%d", argPosition))
		args = append(args, *update.Speed)
		argPosition++
	}

	if update.Heading != nil {
		insertColumns = append(insertColumns, "heading")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("heading = $%d", argPosition))
		args = append(args, *update.Heading)
		argPosition++
	}

	if update.SignalStrength != nil {
		insertColumns = append(insertColumns, "signal_strength")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("signal_strength = $%d", argPosition))
		args = append(args, *update.SignalStrength)
		argPosition++
	}

	if update.Battery != nil {
		insertColumns = append(insertColumns, "battery")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("battery = $%d", argPosition))
		args = append(args, *update.Battery)
		argPosition++
	}

	if update.Armed != nil {
		insertColumns = append(insertColumns, "armed")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("armed = $%d", argPosition))
		args = append(args, *update.Armed)
		argPosition++
	}

	if update.FlightMode != nil {
		insertColumns = append(insertColumns, "flight_mode")
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", argPosition))
		setClauses = append(setClauses, fmt.Sprintf("flight_mode = $%d", argPosition))
		args = append(args, *update.FlightMode)
		argPosition++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, update.DeviceID)
	deviceIDPlaceholder := fmt.Sprintf("$%d", argPosition)

	query := fmt.Sprintf(`
        INSERT INTO device_telemetry (device_id, %s, updated_at)
        VALUES (%s, %s, NOW())
        ON CONFLICT (device_id)
        DO UPDATE SET %s
    `,
		strings.Join(insertColumns, ", "),
		deviceIDPlaceholder,
		strings.Join(insertPlaceholders, ", "),
		strings.Join(setClauses, ", "),
	)

	_, err := db.Exec(query, args...)
	return err
}
