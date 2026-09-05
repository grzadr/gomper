package scanner

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed profiles/*.gitignore
var embeddedProfilesFS embed.FS

type profilesFileSystem interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// profilesFS holds the filesystem for embedded profiles, allowing test overrides.
var profilesFS profilesFileSystem = embeddedProfilesFS

// ListProfiles returns a slice of available embedded ignore profile names.
func ListProfiles() ([]string, error) {
	entries, err := fs.ReadDir(profilesFS, "profiles")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded profiles directory: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gitignore") {
			name := strings.TrimSuffix(entry.Name(), ".gitignore")
			profiles = append(profiles, name)
		}
	}
	return profiles, nil
}

// LoadProfilePatterns loads and converts gitignore patterns from an embedded profile into regex strings.
func LoadProfilePatterns(profileName string) ([]string, error) {
	cleanName := strings.ToLower(strings.TrimSpace(profileName))
	filePath := "profiles/" + cleanName + ".gitignore"

	data, err := profilesFS.ReadFile(filePath)
	if err != nil {
		available, _ := ListProfiles()
		return nil, fmt.Errorf("unknown ignore profile %q (available profiles: %s)", profileName, strings.Join(available, ", "))
	}

	return ParseGitignoreContent(string(data)), nil
}

// ParseGitignoreContent converts gitignore text file contents into a slice of regex pattern strings.
func ParseGitignoreContent(content string) []string {
	lines := strings.Split(content, "\n")
	var regexes []string

	for _, line := range lines {
		regexPattern := GitignoreToRegex(line)
		if regexPattern != "" {
			regexes = append(regexes, regexPattern)
		}
	}

	return regexes
}

// GitignoreToRegex converts a single gitignore glob line into a regular expression pattern string.
func GitignoreToRegex(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
		return ""
	}

	isAnchored := strings.HasPrefix(line, "/")

	// Strip leading and trailing slashes for path matching
	line = strings.TrimPrefix(line, "/")
	line = strings.TrimSuffix(line, "/")

	var sb strings.Builder
	for i := range len(line) {
		ch := line[i]
		switch ch {
		case '*':
			sb.WriteString(".*")
		case '?':
			sb.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(ch)
		default:
			sb.WriteByte(ch)
		}
	}

	pattern := sb.String()
	if pattern == "" {
		return ""
	}

	if isAnchored {
		return "^" + pattern + "(/|$)"
	}
	return "(^|/)" + pattern + "(/|$)"
}

