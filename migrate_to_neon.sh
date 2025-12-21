#!/bin/bash
set -e

# Configuration
VM_IP="4.247.135.200"
KEY_FILE="./vm_key.pem"
REMOTE_USER="azureuser"

# Neon Connection String
NEON_DB_URL="postgresql://neondb_owner:npg_JtK2CjE7kRNv@ep-crimson-scene-ahxj1tja-pooler.c-3.us-east-1.aws.neon.tech/neondb?sslmode=require"

echo ">>> Starting Migration (Remote Execution on Azure VM)..."

# We run the entire pipeline ON the VM to use its Docker and bandwidth
ssh -o StrictHostKeyChecking=no -i $KEY_FILE $REMOTE_USER@$VM_IP "bash -s" <<EOF
  set -e
  echo ">>> [Remote] Dump and Restore starting..."
  
  # 1. Create Dump from running DB container
  # We use 'docker-compose exec' to run pg_dump inside the container
  cd ~/app
  
  echo ">>> [Remote] Piping dump to Neon..."
  # We pipe the output of pg_dump directly into a temporary postgres container running psql connected to Neon
  # We use 'sudo' for docker commands as per the setup
  
  sudo docker-compose exec -T db pg_dump -U vyom -d vyom --clean --if-exists --no-owner --no-privileges | \\
  sudo docker run --rm -i postgres:15 psql "$NEON_DB_URL"
  
  echo ">>> [Remote] Migration completed successfully!"
EOF

echo ">>> Local script finished."
