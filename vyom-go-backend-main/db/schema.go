package db

import "log"

func InitSchema() error {
	schema := `
    CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        clerk_user_id VARCHAR(255) UNIQUE,
        email VARCHAR(255),
        created_at TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS devices (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        public_key TEXT UNIQUE NOT NULL,
        pairing_code INT UNIQUE,
        owner_id UUID REFERENCES users(id),
        name VARCHAR(255),
        status VARCHAR(20) DEFAULT 'StandBy',
        node_id VARCHAR(20),
        last_seen TIMESTAMP,
        created_at TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS collaborators (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        email VARCHAR(255),
        added_at TIMESTAMP DEFAULT NOW(),
        UNIQUE(device_id, user_id, email)
    );

    CREATE TABLE IF NOT EXISTS device_telemetry (
        device_id UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
        latitude FLOAT,
        longitude FLOAT,
        altitude FLOAT,          
        speed FLOAT,              
        heading FLOAT,           
        signal_strength INT,     
        battery INT,
        updated_at TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS zerotier_config (
        device_id UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
        zerotier_ip VARCHAR(50),

        created_at TIMESTAMP DEFAULT NOW(),
        updated_at TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS livekit_rooms (
        device_id UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
        room_name VARCHAR(255) UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS challenges (
        device_id UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
        challenge TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT NOW(),
        expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '5 minutes'
    );
    `

	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Migration for new columns
	migration := `
    ALTER TABLE device_telemetry ADD COLUMN IF NOT EXISTS armed BOOLEAN;
    ALTER TABLE device_telemetry ADD COLUMN IF NOT EXISTS flight_mode VARCHAR(50);
    `
	_, err = DB.Exec(migration)
	if err != nil {
		log.Printf("⚠️ Migration warning: %v", err)
		// Don't fail hard on migration if it's just column exists issue, though IF NOT EXISTS handles it.
	}

	log.Println("✅ Schema initialized!")
	return nil
}
