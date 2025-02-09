package main

import (
	"fmt"
	"image"
	"image/color"
	"net"
	"time"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/host/v3"
)

const (
	width  = 128
	height = 64
)

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

// getEth0IPv4 gets the IPv4 address of eth0
func getEth0IPv4() string {
	iface, err := net.InterfaceByName("eth0")
	if err != nil {
		return "No eth0"
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "No IP"
	}

	for _, addr := range addrs {
		// Check if the address is an IP network
		if ipnet, ok := addr.(*net.IPNet); ok {
			// Check if it's an IPv4 address
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "No IPv4"
}

func main() {
	// Initialize periph.io
	if _, err := host.Init(); err != nil {
		panic(fmt.Sprintf("failed to initialize periph: %v", err))
	}

	// Open default I2C bus
	bus, err := i2creg.Open("")
	if err != nil {
		panic(fmt.Sprintf("failed to open I2C: %v", err))
	}
	defer bus.Close()

	// Create new SSD1306 device
	dev, err := ssd1306.NewI2C(bus, &ssd1306.Opts{
		W: width,
		H: height,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize SSD1306: %v", err))
	}

	// Create a new image buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Main update loop
	for {
		// Clear the image
		for i := 0; i < width*height*4; i++ {
			img.Pix[i] = 0
		}

		// Get IP address
		ipAddr := getEth0IPv4()

		// Get CPU usage
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			panic(fmt.Sprintf("failed to get CPU usage: %v", err))
		}

		// Get memory usage
		memInfo, err := mem.VirtualMemory()
		if err != nil {
			panic(fmt.Sprintf("failed to get memory info: %v", err))
		}

		// Draw IP address
		addLabel(img, 5, 12, fmt.Sprintf("IP: %s", ipAddr))

		// Draw CPU usage
		addLabel(img, 5, 27, "CPU")
		drawBar(img, 35, 20, 88, 10, cpuPercent[0]/100.0)
		addLabel(img, 5, 40, fmt.Sprintf("%.1f%%", cpuPercent[0]))

		// Draw memory usage
		addLabel(img, 5, 55, "MEM")
		drawBar(img, 35, 48, 88, 10, float64(memInfo.UsedPercent)/100.0)
		addLabel(img, 5, 63, fmt.Sprintf("%.1f%%", memInfo.UsedPercent))

		// Update the display
		if err := dev.Draw(img.Bounds(), img, image.Point{0, 0}); err != nil {
			panic(fmt.Sprintf("failed to draw to display: %v", err))
		}

		// Wait before next update
		time.Sleep(1 * time.Second)
	}
}

