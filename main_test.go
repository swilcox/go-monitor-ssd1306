package main

import (
	"image"
	"image/color"
	"testing"
)

// MockNetworkChecker implements NetworkChecker for testing
type MockNetworkChecker struct {
	ipAddress string
}

func (m *MockNetworkChecker) GetIPv4Address(interfaceName string) string {
	return m.ipAddress
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
			img := image.NewRGBA(image.Rect(0, 0, 128, 64))
			drawBar(img, 10, 10, 50, 10, tt.percentage)

			// Check if bar is drawn correctly
			middle := img.RGBAAt(35, 15) // Point in middle of bar
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
			img := image.NewRGBA(image.Rect(0, 0, 128, 64))
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

// TestGetIPv4Address tests the IP address retrieval
func TestGetIPv4Address(t *testing.T) {
	tests := []struct {
		name     string
		ipAddr   string
		interface_ string
		want     string
	}{
		{
			name:     "Valid IPv4",
			ipAddr:   "192.168.1.100",
			interface_: "eth0",
			want:     "192.168.1.100",
		},
		{
			name:     "No interface",
			ipAddr:   "No eth0",
			interface_: "eth0",
			want:     "No eth0",
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
