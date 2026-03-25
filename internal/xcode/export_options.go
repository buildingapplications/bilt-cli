package xcode

import (
	"fmt"
	"os"
	"path/filepath"
)

const exportOptionsTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>method</key>
    <string>development</string>
    <key>signingStyle</key>
    <string>automatic</string>
    <key>compileBitcode</key>
    <false/>
    <key>stripSwiftSymbols</key>
    <true/>
    <key>teamID</key>
    <string>%s</string>
</dict>
</plist>`

// WriteExportOptions writes an ExportOptions.plist for development signing.
func WriteExportOptions(buildDir, teamID string) (string, error) {
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("creating build directory: %w", err)
	}

	plistPath := filepath.Join(buildDir, "ExportOptions.plist")
	content := fmt.Sprintf(exportOptionsTmpl, teamID)

	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing ExportOptions.plist: %w", err)
	}
	return plistPath, nil
}
