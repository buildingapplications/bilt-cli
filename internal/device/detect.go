package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// Device represents a connected iOS device.
type Device struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	UDID       string `json:"udid"`
	IOSVersion string `json:"ios_version"`
	Connection string `json:"connection"` // "USB" or "WiFi"
}

// devicectlOutput matches the JSON structure from xcrun devicectl list devices.
type devicectlOutput struct {
	Result struct {
		Devices []devicectlDevice `json:"devices"`
	} `json:"result"`
}

type devicectlDevice struct {
	DeviceProperties struct {
		Name        string `json:"name"`
		OSVersion   string `json:"osVersionNumber"`
		ProductType string `json:"productType"`
	} `json:"deviceProperties"`
	HardwareProperties struct {
		ProductType string `json:"productType"`
		Platform    string `json:"platform"`
	} `json:"hardwareProperties"`
	ConnectionProperties struct {
		TransportType string `json:"transportType"`
	} `json:"connectionProperties"`
	Identifier string `json:"identifier"`
}

// Detect returns a list of connected iOS devices.
func Detect(ctx context.Context, r *runner.Runner) ([]Device, error) {
	devices, err := detectViaDevicectl(ctx, r)
	if err == nil && len(devices) > 0 {
		return devices, nil
	}

	// Fallback to ios-deploy
	return detectViaIOSDeploy(ctx, r)
}

func detectViaDevicectl(ctx context.Context, r *runner.Runner) ([]Device, error) {
	stdout, _, err := r.Run(ctx, "", "xcrun", "devicectl", "list", "devices", "--json-output", "/dev/stdout")
	if err != nil {
		return nil, fmt.Errorf("devicectl: %w", err)
	}

	// devicectl writes a human-readable table before the JSON.
	// Extract the JSON by finding the first '{'.
	jsonStr := stdout
	if idx := strings.Index(stdout, "{"); idx >= 0 {
		jsonStr = stdout[idx:]
	}

	var output devicectlOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, fmt.Errorf("parsing devicectl output: %w", err)
	}

	var devices []Device
	for _, d := range output.Result.Devices {
		// Skip simulators and non-iOS
		if d.HardwareProperties.Platform != "" && d.HardwareProperties.Platform != "iOS" {
			continue
		}

		conn := "USB"
		if strings.Contains(strings.ToLower(d.ConnectionProperties.TransportType), "wifi") {
			conn = "WiFi"
		}

		model := d.HardwareProperties.ProductType
		if model == "" {
			model = d.DeviceProperties.ProductType
		}

		devices = append(devices, Device{
			Name:       d.DeviceProperties.Name,
			Model:      friendlyModel(model),
			UDID:       d.Identifier,
			IOSVersion: d.DeviceProperties.OSVersion,
			Connection: conn,
		})
	}
	return devices, nil
}

func detectViaIOSDeploy(ctx context.Context, r *runner.Runner) ([]Device, error) {
	stdout, _, err := r.Run(ctx, "", "ios-deploy", "--detect", "--timeout", "5")
	if err != nil {
		return nil, fmt.Errorf("ios-deploy: %w", err)
	}

	var devices []Device
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "Found") {
			continue
		}
		// ios-deploy output: "Found <UDID> ... '<Name>'"
		// Parse UDID and name from the line
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		udid := parts[1]

		name := "Unknown Device"
		if idx := strings.Index(line, "'"); idx >= 0 {
			end := strings.LastIndex(line, "'")
			if end > idx {
				name = line[idx+1 : end]
			}
		}

		devices = append(devices, Device{
			Name:       name,
			UDID:       udid,
			Connection: "USB",
		})
	}
	return devices, nil
}

// friendlyModel converts product types like "iPhone16,1" to friendlier names.
func friendlyModel(productType string) string {
	// Just return the product type — a full mapping would be huge and stale quickly.
	// Users see their device name which is more useful anyway.
	if productType == "" {
		return "iPhone"
	}
	return productType
}
