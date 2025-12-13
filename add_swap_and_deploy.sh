#!/bin/bash
set -e

VM_IP="4.247.135.200"
KEY_FILE="./vm_key.pem"
REMOTE_USER="azureuser"

echo ">>> Adding Swap Space to VM at $VM_IP to prevent OOM..."

ssh -o StrictHostKeyChecking=no -i $KEY_FILE $REMOTE_USER@$VM_IP << 'EOF'
  # Check if swap exists
  if swapon --show | grep -q "swap"; then
      echo "Swap already exists. Skipping."
  else
      echo "Allocating 2G Swap File..."
      sudo fallocate -l 2G /swapfile
      sudo chmod 600 /swapfile
      sudo mkswap /swapfile
      sudo swapon /swapfile
      echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
      echo "Swap enabled!"
      free -h
  fi
EOF

echo ">>> Swap Added. Retrying Deployment..."
./deploy_to_vm.sh
