package main

import (
	"fmt"
	"image"
	"os"
	"testing"
	"time"
)

// MockDisplay implements the DisplayDevice interface for testing
type MockDisplay struct {
	contrast  uint8
	inverted  bool
	lastImage *image.RGBA
	t         *testing.T  // for debug output
}

func NewMockDisplay(t *testing.T) *MockDisplay {
	return &MockDisplay{
		contrast: 255,
		t:        t,
	}
}

func (d *MockDisplay) SetContrast(contrast uint8) error {
	d.contrast = contrast
	return nil
}

func (d *MockDisplay) Invert(inverted bool) error {
	d.inverted = inverted
	return nil
}

func (d *MockDisplay) Draw(r image.Rectangle, src image.Image, sp image.Point) error {
	d.t.Logf("Draw called with bounds: %v", r)
	if src == nil {
		d.t.Log("Draw called with nil source image")
		return fmt.Errorf("nil source image")
	}
	
	if rgba, ok := src.(*image.RGBA); ok {
		d.t.Logf("Draw called with RGBA image of size: %v", rgba.Bounds())
		d.lastImage = rgba
	} else {
		d.t.Logf("Draw called with non-RGBA image type: %T", src)
		return fmt.Errorf("non-RGBA image")
	}
	return nil
}

func (d *MockDisplay) Halt() error {
	return nil
}

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
			middle := img.RGBAAt(35, 13) // Point in middle of bar
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
		})
	}
}

// TestDisplayManager tests the display manager functionality
func TestDisplayManager(t *testing.T) {
	// Create a temporary config file for testing
	configYAML := []byte(`
screen_duration: 1
invert_duration: 2
day_start_hour: 7
night_start_hour: 18
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

	// Create mock display with debug logging
	mockDisplay := NewMockDisplay(t)

	// Create mock time function
	mockTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local) // noon
	timeNow := func() time.Time {
		return mockTime
	}

	// Create display manager with mocks
	checker := &MockNetworkChecker{ipAddress: "192.168.1.100"}
	dm := &DisplayManager{
		dev:            mockDisplay,
		networkChecker: checker,
		img:            image.NewRGBA(image.Rect(0, 0, width, height)),
		timeNow:        timeNow,
		config: Config{
			DayStartHour:   7,
			NightStartHour: 18,
			NetworkInterface: "eth0",
			ScreenDuration: 5,
			Screens: []Screen{
				{
					Name: "Test Screen",
					Components: []Component{
						{
							Type:   "ip",
							X:      5,
							Y:      20,
							Label:  "IP",
						},
					},
				},
			},
		},
	}

	// Test screen rendering
	t.Log("Testing screen rendering...")
	if err := dm.renderCurrentScreen(); err != nil {
		t.Errorf("Failed to render screen: %v", err)
	}

	if mockDisplay.lastImage == nil {
		t.Error("Expected image to be drawn after renderCurrentScreen")
	}

	// Clear the mock display state
	mockDisplay.lastImage = nil

	// Test individual component rendering
	t.Log("Testing component rendering...")
	comp := Component{
		Type:   "ip",
		X:      5,
		Y:      20,
		Label:  "IP",
	}

	if err := dm.renderComponent(comp); err != nil {
		t.Errorf("Failed to render component: %v", err)
	}

	// Manually draw to display after component render
	t.Log("Testing manual draw...")
	if err := dm.dev.Draw(dm.img.Bounds(), dm.img, image.Point{0, 0}); err != nil {
		t.Errorf("Failed to draw to display: %v", err)
	}

	if mockDisplay.lastImage == nil {
		t.Error("Expected image to be drawn after manual Draw")
	}
}

// MockNetworkChecker implements NetworkChecker for testing
type MockNetworkChecker struct {
	ipAddress string
}

func (m *MockNetworkChecker) GetIPv4Address(interfaceName string) string {
	return m.ipAddress
}

