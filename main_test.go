package main

import (
	"image"
	"image/color"
	"os"
	"testing"
)

// TestDrawBar tests the drawBar function
func TestDrawBar(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		wantEmpty  bool
		wantFull   bool
	}{
		{
			name:       "Empty bar",
			percentage: 0.0,
			wantEmpty:  true,
			wantFull:   false,
		},
		{
			name:       "Full bar",
			percentage: 1.0,
			wantEmpty:  false,
			wantFull:   true,
		},
		{
			name:       "Half bar",
			percentage: 0.5,
			wantEmpty:  false,
			wantFull:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, width, height))
			drawBar(img, 10, 10, 50, barHeight, tt.percentage)

			// Check if bar is drawn correctly
			middle := img.RGBAAt(35, 13) // Point in middle of bar adjusted for new height
			if tt.wantEmpty && middle.R != 0 {
				t.Errorf("Expected empty bar, but got color at middle point")
			}
			if tt.wantFull && middle.R == 0 {
				t.Errorf("Expected full bar, but got no color at middle point")
			}

			// Check border
			border := img.RGBAAt(10, 10) // Top-left corner
			if border.R == 0 {
				t.Errorf("Expected border to be drawn")
			}

			// Verify bar height
			bottomBorder := img.RGBAAt(10, 10+barHeight)
			if bottomBorder.R == 0 {
				t.Errorf("Expected bottom border at correct height")
			}
		})
	}
}

// TestAddLabel tests the text rendering function
func TestAddLabel(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		x, y  int
		empty bool
	}{
		{
			name:  "Simple text",
			text:  "Test",
			x:     10,
			y:     20,
			empty: false,
		},
		{
			name:  "Empty text",
			text:  "",
			x:     10,
			y:     20,
			empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, width, height))
			addLabel(img, tt.x, tt.y, tt.text)

			// Check if any pixels were drawn
			hasPixels := false
			bounds := img.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					if img.RGBAAt(x, y) != (color.RGBA{}) {
						hasPixels = true
						break
					}
				}
			}

			if tt.empty && hasPixels {
				t.Errorf("Expected no pixels for empty text, but found some")
			}
			if !tt.empty && !hasPixels {
				t.Errorf("Expected pixels for non-empty text, but found none")
			}
		})
	}
}

// MockNetworkChecker implements NetworkChecker for testing
type MockNetworkChecker struct {
	ipAddress string
}

func (m *MockNetworkChecker) GetIPv4Address(interfaceName string) string {
	return m.ipAddress
}

// TestGetIPv4Address tests the IP address retrieval
func TestGetIPv4Address(t *testing.T) {
	tests := []struct {
		name       string
		ipAddr     string
		interface_ string
		want       string
	}{
		{
			name:       "Valid IPv4",
			ipAddr:     "192.168.1.100",
			interface_: "eth0",
			want:      "192.168.1.100",
		},
		{
			name:       "No interface",
			ipAddr:     "No eth0",
			interface_: "eth0",
			want:       "No eth0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &MockNetworkChecker{ipAddress: tt.ipAddr}
			got := checker.GetIPv4Address(tt.interface_)
			if got != tt.want {
				t.Errorf("GetIPv4Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDisplayManager tests the display manager functionality
func TestDisplayManager(t *testing.T) {
	// Create a temporary config file for testing
	configYAML := []byte(`
screen_duration: 1
network_interface: eth0
screens:
  - name: Test Screen
    components:
      - type: time
        x: 5
        y: 10
        time_format: "15:04:05"
      - type: ip
        x: 5
        y: 20
        label: IP
`)

	tmpfile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(configYAML); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create display manager with mock network checker
	checker := &MockNetworkChecker{ipAddress: "192.168.1.100"}
	dm, err := NewDisplayManager(tmpfile.Name(), checker)
	if err != nil {
		t.Fatal(err)
	}

	// Test config parsing
	if dm.config.ScreenDuration != 1 {
		t.Errorf("Expected screen duration 1, got %d", dm.config.ScreenDuration)
	}
	if dm.config.NetworkInterface != "eth0" {
		t.Errorf("Expected network interface eth0, got %s", dm.config.NetworkInterface)
	}
	if len(dm.config.Screens) != 1 {
		t.Errorf("Expected 1 screen, got %d", len(dm.config.Screens))
	}

	// Test component parsing
	screen := dm.config.Screens[0]
	if len(screen.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(screen.Components))
	}

	// Test time component
	timeComp := screen.Components[0]
	if timeComp.Type != "time" {
		t.Errorf("Expected time component, got %s", timeComp.Type)
	}
	if timeComp.TimeFormat != "15:04:05" {
		t.Errorf("Expected time format 15:04:05, got %s", timeComp.TimeFormat)
	}
}

// TestTimeComponent tests the time display functionality
func TestTimeComponent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	comp := Component{
		Type:       "time",
		X:          5,
		Y:          10,
		TimeFormat: "15:04:05",
	}

	// Create a display manager with a mock network checker
	dm := &DisplayManager{
		img:           img,
		networkChecker: &MockNetworkChecker{},
	}

	// Test time rendering
	err := dm.renderComponent(comp)
	if err != nil {
		t.Errorf("Failed to render time component: %v", err)
	}

	// Verify that something was drawn
	hasPixels := false
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y) != (color.RGBA{}) {
				hasPixels = true
				break
			}
		}
	}

	if !hasPixels {
		t.Error("Expected time component to draw pixels")
	}
}

