# Vyom Middleware - Raspberry Pi Installation Guide

## 📦 Package Details
- **File**: `vyom-middleware_2.5.0_arm64.deb`
- **Architecture**: `arm64` (Raspberry Pi 3B+, 4, 5 running 64-bit OS)
- **Version**: `2.5.0`

---

## 🚀 Installation Steps

### 1. Transfer Package to RPi
Use `scp` or a USB drive to copy the file to your Raspberry Pi.
```bash
# Replace 'user' and 'pi-ip' with actual values
scp vyom-middleware_2.5.0_arm64.deb user@192.168.1.X:~/
```

### 2. Install Dependencies & Package
SSH into the Raspberry Pi and run:
```bash
ssh user@192.168.1.X

# Update package list
sudo apt-get update

# Install the package
sudo dpkg -i vyom-middleware_2.5.0_arm64.deb

# Fix missing dependencies (if any)
sudo apt-get install -f -y
```

### 3. Verify Installation
Check if the service is running:
```bash
sudo systemctl status vyom-middleware
```
*Output should show `Active: active (running)`.*

---

## ⚙️ Configuration & Usage

### Setup via Web UI
1. Ensure your RPi and Laptop are on the **same network** (or RPi has public IP).
2. Open your browser and navigate to:
   `http://<RPI_IP_ADDRESS>:8085`
3. Follow the instructions to **Claim Device**.

### Hardware Connection
- **Flight Controller**: Connect via USB or Serial.
- **Camera**: Connect via USB or CSI.
- **Restart Middleware**:
  ```bash
  sudo systemctl restart vyom-middleware
  ```

---

## 🔧 Troubleshooting

### "Address already in use"
If the service fails to start, check logs:
```bash
journalctl -u vyom-middleware -f
```
*Note: Version 2.5.0 automatically attempts to kill processes blocking port 8085.*

### Camera Issues
Ensure camera permissions are set:
```bash
sudo usermod -a -G video $USER
```
Check camera detection:
```bash
v4l2-ctl --list-devices
```

### Firewall
If you cannot access the Web UI:
```bash
sudo ufw allow 8085/tcp
```
