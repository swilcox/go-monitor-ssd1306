# Raspberry Pi OLED System Monitor

A Go application for displaying system information on an SSD1306 OLED display connected to a Raspberry Pi. The display shows multiple "screens" of information that rotate automatically, with values updating every second.

## Features

- Display system metrics:
  - CPU usage with progress bar
  - Memory usage with progress bar
  - Disk usage with progress bar
  - IP address for configured network interface
  - Current time in various formats
- Configurable virtual screens that rotate at specified intervals
- Automatic brightness adjustment based on time of day
- Optional display inversion to prevent burn-in
- YAML configuration
- Support for both 128x64 and 128x32 OLED displays (SSD1306)

## Hardware Setup

### Required Components
- Raspberry Pi (any model with I2C support)
- SSD1306 OLED Display (128x64 or 128x32)
- 4 jumper wires

### Wiring Instructions

Connect the OLED display to your Raspberry Pi:

```
OLED Display    Raspberry Pi
VCC/VDD    ->   3.3V (Pin 1)
GND        ->   Ground (Pin 6)
SCL        ->   GPIO 3/SCL (Pin 5)
SDA        ->   GPIO 2/SDA (Pin 3)
```

### Enable I2C on Raspberry Pi

1. Run raspi-config:
```bash
sudo raspi-config
```

2. Navigate to:
   - Interface Options
   - I2C
   - Enable I2C interface

3. Reboot your Raspberry Pi:
```bash
sudo reboot
```

4. Verify I2C is working:
```bash
sudo apt-get install i2c-tools
sudo i2cdetect -y 1
```

You should see your device (typically at address 0x3C).

## Installation

1. Install Go on your Raspberry Pi:
```bash
sudo apt-get update
sudo apt-get install golang
```

2. Clone this repository:
```bash
git clone https://github.com/swilcox/go-monitor-ssd1306.git
cd go-monitor-ssd1306
```

3. Install dependencies:
```bash
go mod tidy
```

4. Build and run:
```bash
go build
./go-monitor-ssd1306
```

## Configuration

The application uses a YAML configuration file (`config.yaml`) to define what information to display and how to display it.

### Example Configuration

```yaml
# Display configuration
screen_duration: 5      # seconds between screen switches
invert_duration: 30     # seconds between display inversion (0 to disable)
day_start_hour: 7      # 7:00 AM - switch to bright mode
night_start_hour: 18   # 6:00 PM - switch to dim mode
network_interface: eth0

screens:
  - name: System Status
    components:
      - type: time
        x: 5
        y: 10
        time_format: "15:04:05"

      - type: ip
        x: 5
        y: 22
        label: IP

      - type: cpu
        x: 5
        y: 34
        label: CPU
        show_bar: true
        bar_width: 88

      - type: memory
        x: 5
        y: 49
        label: MEM
        show_bar: true
        bar_width: 88

  - name: Storage
    components:
      - type: time
        x: 5
        y: 10
        time_format: "15:04"

      - type: disk
        x: 5
        y: 25
        label: Disk
        show_bar: true
        bar_width: 118
```

### Configuration Options

#### Global Settings
- `screen_duration`: Time in seconds before switching to next screen
- `invert_duration`: Time in seconds between display inversion toggles (set to 0 to disable)
- `network_interface`: Network interface to monitor for IP address
- `day_start_hour`: Hour (0-23) to switch to bright mode
- `night_start_hour`: Hour (0-23) to switch to dim mode

#### Component Types
1. Time Component:
   ```yaml
   type: time
   x: 5           # X position
   y: 10          # Y position
   time_format: "15:04:05"  # Go time format string
   ```
   Available time formats:
   - "15:04:05" - 24-hour with seconds
   - "15:04" - 24-hour without seconds
   - "3:04 PM" - 12-hour without seconds
   - "3:04:05 PM" - 12-hour with seconds
   - "Mon 15:04" - Day and time
   - "02-Jan" - Date

2. System Metrics (CPU, Memory, Disk):
   ```yaml
   type: cpu    # or memory, disk
   x: 5
   y: 25
   label: "CPU"
   show_bar: true
   bar_width: 88
   ```

3. IP Address:
   ```yaml
   type: ip
   x: 5
   y: 22
   label: "IP"
   ```

### Display Behavior
- All component values update every second
- Screens rotate based on `screen_duration`
- Display brightness automatically adjusts based on time of day
- Optional display inversion helps prevent burn-in
- Progress bars are 7 pixels high

## Running as a Service

To run the monitor at startup, create a systemd service:

1. Create service file:
```bash
sudo nano /etc/systemd/system/oled-monitor.service
```

2. Add the following content:
```ini
[Unit]
Description=OLED System Monitor
After=network.target

[Service]
ExecStart=/path/to/go-monitor-ssd1306
WorkingDirectory=/path/to/go-monitor-ssd1306
StandardOutput=inherit
StandardError=inherit
Restart=always
User=pi

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:
```bash
sudo systemctl enable oled-monitor
sudo systemctl start oled-monitor
```

## Contributing

Contributions are welcome! Feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

