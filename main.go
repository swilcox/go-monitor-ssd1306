package main

import (
	"fmt"
	"image"
	"image/color"
	"net"
	"os"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"gopkg.in/yaml.v3"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/host/v3"
)

const (
	width          = 128
	height         = 64
	barHeight      = 7
	brightContrast = 255
	dimContrast    = 1
	tempFile       = "/sys/class/thermal/thermal_zone0/temp"
)

// Config represents the main configuration
type Config struct {
	ScreenDuration   int      `yaml:"screen_duration"`
	NetworkInterface string   `yaml:"network_interface"`
	InvertDuration   int      `yaml:"invert_duration"`  // seconds between invert toggles, 0 to disable
	DayStartHour     int      `yaml:"day_start_hour"`   // hour to switch to bright mode (0-23)
	NightStartHour   int      `yaml:"night_start_hour"` // hour to switch to dim mode (0-23)
	Screens          []Screen `yaml:"screens"`
}

// Screen represents a single virtual screen configuration
type Screen struct {
	Name       string      `yaml:"name"`
	Components []Component `yaml:"components"`
}

// Component represents a display component configuration
type Component struct {
	Type       string `yaml:"type"`
	X          int    `yaml:"x"`
	Y          int    `yaml:"y"`
	Label      string `yaml:"label,omitempty"`
	ShowBar    bool   `yaml:"show_bar,omitempty"`
	BarWidth   int    `yaml:"bar_width,omitempty"`
	TimeFormat string `yaml:"time_format,omitempty"`
}

// NetworkChecker interface for getting IP addresses
type NetworkChecker interface {
	GetIPv4Address(interfaceName string) string
}

// RealNetworkChecker implements NetworkChecker for actual network interfaces
type RealNetworkChecker struct{}

// GetIPv4Address gets the IPv4 address of the specified interface
func (r *RealNetworkChecker) GetIPv4Address(interfaceName string) string {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return fmt.Sprintf("No %s", interfaceName)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "No IP"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "No IPv4"
}

// DisplayDevice interface defines the methods we need from a display
type DisplayDevice interface {
	SetContrast(contrast uint8) error
	Invert(inverted bool) error
	Draw(r image.Rectangle, src image.Image, sp image.Point) error
	Halt() error
}

// DisplayManager handles screen rotation and rendering
type DisplayManager struct {
	config         Config
	currentScreen  int
	networkChecker NetworkChecker
	dev            DisplayDevice
	img            *image.RGBA
	isInverted     bool
	timeNow        func() time.Time
}

// addLabel adds a text label to the image
func addLabel(img *image.RGBA, x, y int, label string) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

// drawBar draws a horizontal progress bar
func drawBar(img *image.RGBA, x, y, width, height int, percentage float64) {
	// Draw border
	for i := x; i < x+width; i++ {
		img.Set(i, y, color.White)
		img.Set(i, y+height, color.White)
	}
	for i := y; i < y+height; i++ {
		img.Set(x, i, color.White)
		img.Set(x+width, i, color.White)
	}

	// Fill bar based on percentage
	fillWidth := int(float64(width-2) * percentage)
	for i := x + 1; i < x+1+fillWidth; i++ {
		for j := y + 1; j < y+height; j++ {
			img.Set(i, j, color.White)
		}
	}
}

func NewDisplayManager(configPath string, networkChecker NetworkChecker) (*DisplayManager, error) {
	// Read configuration
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Initialize display
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize periph: %v", err)
	}

	bus, err := i2creg.Open("")
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C: %v", err)
	}

	dev, err := ssd1306.NewI2C(bus, &ssd1306.Opts{
		W: width,
		H: height,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SSD1306: %v", err)
	}

	return &DisplayManager{
		config:         config,
		currentScreen:  0,
		networkChecker: networkChecker,
		dev:            dev,
		img:            image.NewRGBA(image.Rect(0, 0, width, height)),
		timeNow:        time.Now,
	}, nil
}

func (dm *DisplayManager) updateBrightness() error {
	hour := dm.timeNow().Hour()
	isDaytime := hour >= dm.config.DayStartHour && hour < dm.config.NightStartHour

	contrast := dimContrast
	if isDaytime {
		contrast = brightContrast
	}

	return dm.dev.SetContrast(uint8(contrast))
}

func (dm *DisplayManager) Run() error {
	screenTicker := time.NewTicker(time.Duration(dm.config.ScreenDuration) * time.Second)
	defer screenTicker.Stop()

	// Update values every second
	updateTicker := time.NewTicker(1 * time.Second)
	defer updateTicker.Stop()

	var invertTicker *time.Ticker
	var invertChan <-chan time.Time
	if dm.config.InvertDuration > 0 {
		invertTicker = time.NewTicker(time.Duration(dm.config.InvertDuration) * time.Second)
		defer invertTicker.Stop()
		invertChan = invertTicker.C
	}

	// Initialize brightness based on current time
	if err := dm.updateBrightness(); err != nil {
		return fmt.Errorf("failed to set initial brightness: %v", err)
	}

	// Check brightness every minute
	brightnessTicker := time.NewTicker(1 * time.Minute)
	defer brightnessTicker.Stop()

	// Render initial screen
	if err := dm.renderCurrentScreen(); err != nil {
		return err
	}

	for {
		select {
		case <-screenTicker.C:
			dm.currentScreen = (dm.currentScreen + 1) % len(dm.config.Screens)
			if err := dm.renderCurrentScreen(); err != nil {
				return err
			}

		case <-updateTicker.C:
			if err := dm.renderCurrentScreen(); err != nil {
				return err
			}

		case <-invertChan:
			dm.isInverted = !dm.isInverted
			if err := dm.dev.Invert(dm.isInverted); err != nil {
				return fmt.Errorf("failed to toggle invert: %v", err)
			}

		case <-brightnessTicker.C:
			if err := dm.updateBrightness(); err != nil {
				return fmt.Errorf("failed to update brightness: %v", err)
			}
		}
	}
}

func (dm *DisplayManager) renderCurrentScreen() error {
	// Clear the image
	for i := 0; i < width*height*4; i++ {
		dm.img.Pix[i] = 0
	}

	screen := dm.config.Screens[dm.currentScreen]
	for _, comp := range screen.Components {
		if err := dm.renderComponent(comp); err != nil {
			return fmt.Errorf("error rendering component: %v", err)
		}
	}

	return dm.dev.Draw(dm.img.Bounds(), dm.img, image.Point{0, 0})
}

func (dm *DisplayManager) renderComponent(comp Component) error {
	switch comp.Type {
	case "time":
		timeFormat := comp.TimeFormat
		if timeFormat == "" {
			timeFormat = "15:04:05" // default to 24-hour time with seconds
		}
		currentTime := time.Now().Format(timeFormat)
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s%s",
			func() string {
				if comp.Label != "" {
					return comp.Label + ": "
				}
				return ""
			}(),
			currentTime))

	case "ip":
		ipAddr := dm.networkChecker.GetIPv4Address(dm.config.NetworkInterface)
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s: %s", comp.Label, ipAddr))

	case "cpu":
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			return err
		}
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s: %.1f%%", comp.Label, cpuPercent[0]))
		if comp.ShowBar {
			drawBar(dm.img, comp.X, comp.Y+5, comp.BarWidth, barHeight, cpuPercent[0]/100.0)
		}

	case "memory":
		memInfo, err := mem.VirtualMemory()
		if err != nil {
			return err
		}
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s: %.1f%%", comp.Label, memInfo.UsedPercent))
		if comp.ShowBar {
			drawBar(dm.img, comp.X, comp.Y+5, comp.BarWidth, barHeight, float64(memInfo.UsedPercent)/100.0)
		}

	case "disk":
		usage, err := disk.Usage("/")
		if err != nil {
			return err
		}
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s: %.1f%%", comp.Label, usage.UsedPercent))
		if comp.ShowBar {
			drawBar(dm.img, comp.X, comp.Y+5, comp.BarWidth, barHeight, float64(usage.UsedPercent)/100.0)
		}

	case "temperature":
		temp, err := os.ReadFile(tempFile)
		if err != nil {
			return fmt.Errorf("failed to read temperature: %v", err)
		}
		tempValue := string(temp)
		if len(tempValue) > 0 {
			tempValue = tempValue[:len(tempValue)-1] // Remove newline
		}
		tempCelsius := float64(0)
		if _, err := fmt.Sscanf(tempValue, "%f", &tempCelsius); err != nil {
			return fmt.Errorf("failed to parse temperature: %v", err)
		}
		tempCelsius /= 1000.0 // Convert to Celsius
		addLabel(dm.img, comp.X, comp.Y, fmt.Sprintf("%s: %.1f C", comp.Label, tempCelsius))
		if comp.ShowBar {
			drawBar(dm.img, comp.X, comp.Y+5, comp.BarWidth, barHeight, tempCelsius/100.0)
		}

	}

	return nil
}

func main() {
	networkChecker := &RealNetworkChecker{}
	dm, err := NewDisplayManager("config.yaml", networkChecker)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize display manager: %v", err))
	}

	if err := dm.Run(); err != nil {
		panic(fmt.Sprintf("display manager error: %v", err))
	}
}
