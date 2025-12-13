#!/bin/bash
set -e

VM_IP="4.247.135.200"
KEY_FILE="./vm_key.pem"
REMOTE_USER="azureuser"
REMOTE_DIR="/home/azureuser/app"

echo ">>> Preparing VM at $VM_IP..."

# 1. Install Docker (If not exists)
ssh -o StrictHostKeyChecking=no -i $KEY_FILE $REMOTE_USER@$VM_IP << 'EOF'
  if ! command -v docker &> /dev/null; then
      echo "Installing Docker..."
      sudo apt-get update
      sudo apt-get install -y docker.io docker-compose
      sudo usermod -aG docker $USER
      echo "Docker installed. Please re-login or wait..."
  fi
  mkdir -p ~/app
EOF

echo ">>> Installing Docker completed (or already present)."

# 2. Upload Code (Excluding large files if any - simple scp for now)
echo ">>> Uploading Backend Code..."
# We zip it to transfer faster and cleaner
tar -czf backend_deploy_package.tar.gz -C vyom-go-backend-main .

scp -o StrictHostKeyChecking=no -i $KEY_FILE backend_deploy_package.tar.gz $REMOTE_USER@$VM_IP:$REMOTE_DIR/

# 3. Upload Env File
echo ">>> Uploading Secrets..."
# We copy backend/.env to the remote as .env.backend
scp -o StrictHostKeyChecking=no -i $KEY_FILE .env $REMOTE_USER@$VM_IP:$REMOTE_DIR/.env.backend

# 4. Deploy
echo ">>> Deploying Services..."
ssh -o StrictHostKeyChecking=no -i $KEY_FILE $REMOTE_USER@$VM_IP << 'EOF'
  cd ~/app
  tar -xzf backend_deploy_package.tar.gz
  rm backend_deploy_package.tar.gz
  
  # Ensure clean restart
  sudo docker-compose down || true
  # Force rebuild to pick up code changes (ignore cache)
  sudo docker-compose build --no-cache
  sudo docker-compose up -d
  
  echo ">>> DEPLOYMENT SUCCESSFUL!"
  echo ">>> Backend running on port 8080"
EOF

# Cleanup
rm backend_deploy_package.tar.gz
