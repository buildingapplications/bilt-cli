package xcode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// XcodeTeam represents a team registered in Xcode's account preferences.
type XcodeTeam struct {
	TeamID   string
	TeamName string
	TeamType string
	IsFree   bool
}

// FindXcodeTeams reads the provisioning teams from Xcode's preferences.
// These are the teams that xcodebuild CLI can actually use.
// The team IDs here may differ from the signing certificate team IDs for free accounts.
func FindXcodeTeams() ([]XcodeTeam, error) {
	// Use `defaults read` which outputs NeXT-style plist text
	out, err := exec.Command("defaults", "read", "com.apple.dt.Xcode", "IDEProvisioningTeamByIdentifier").Output()
	if err != nil {
		return nil, fmt.Errorf("reading Xcode provisioning teams: %w", err)
	}

	return parseXcodeTeams(string(out)), nil
}

// parseXcodeTeams parses the NeXT-style plist output from `defaults read`.
// Format:
//
//	{
//	    "UUID" = (
//	        {
//	            isFreeProvisioningTeam = 1;
//	            teamID = XXXXXXXXXX;
//	            teamName = "Name (Personal Team)";
//	            teamType = "Personal Team";
//	        }
//	    );
//	}
func parseXcodeTeams(output string) []XcodeTeam {
	var teams []XcodeTeam
	seen := make(map[string]bool)

	lines := strings.Split(output, "\n")
	var currentTeam *XcodeTeam

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSuffix(trimmed, ";")

		switch {
		case strings.Contains(trimmed, "teamID"):
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				if currentTeam == nil {
					currentTeam = &XcodeTeam{}
				}
				currentTeam.TeamID = strings.TrimSpace(parts[1])
			}
		case strings.Contains(trimmed, "teamName"):
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				if currentTeam == nil {
					currentTeam = &XcodeTeam{}
				}
				name := strings.TrimSpace(parts[1])
				name = strings.Trim(name, `"`)
				currentTeam.TeamName = name
			}
		case strings.Contains(trimmed, "teamType"):
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				if currentTeam == nil {
					currentTeam = &XcodeTeam{}
				}
				typ := strings.TrimSpace(parts[1])
				typ = strings.Trim(typ, `"`)
				currentTeam.TeamType = typ
			}
		case strings.Contains(trimmed, "isFreeProvisioningTeam"):
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				if currentTeam == nil {
					currentTeam = &XcodeTeam{}
				}
				currentTeam.IsFree = strings.TrimSpace(parts[1]) == "1"
			}
		}

		// End of a team block
		if trimmed == "}" || trimmed == "}," {
			if currentTeam != nil && currentTeam.TeamID != "" && !seen[currentTeam.TeamID] {
				seen[currentTeam.TeamID] = true
				teams = append(teams, *currentTeam)
			}
			currentTeam = nil
		}
	}

	return teams
}

// PatchTeamID rewrites DEVELOPMENT_TEAM in the .pbxproj file to match the selected team.
// This is necessary because expo prebuild / project templates may hardcode a different team ID,
// and xcodebuild CLI build setting overrides don't fully take effect for provisioning.
func PatchTeamID(projectDir, teamID string) error {
	iosDir := filepath.Join(projectDir, "ios")
	entries, err := os.ReadDir(iosDir)
	if err != nil {
		return nil // no ios dir, nothing to patch
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xcodeproj") {
			continue
		}
		pbxproj := filepath.Join(iosDir, e.Name(), "project.pbxproj")
		data, err := os.ReadFile(pbxproj)
		if err != nil {
			continue
		}

		content := string(data)
		// Replace all DEVELOPMENT_TEAM = <any>; with the correct team
		lines := strings.Split(content, "\n")
		changed := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "DEVELOPMENT_TEAM") && strings.Contains(trimmed, "=") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				newLine := indent + fmt.Sprintf("DEVELOPMENT_TEAM = %s;", teamID)
				if lines[i] != newLine {
					lines[i] = newLine
					changed = true
				}
			}
		}

		if changed {
			if err := os.WriteFile(pbxproj, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return fmt.Errorf("writing pbxproj: %w", err)
			}
		}
	}
	return nil
}
