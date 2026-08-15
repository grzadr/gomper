package scanner

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

// AuxiliaryExtensions contains common template and example suffixes.
var AuxiliaryExtensions = map[string]struct{}{
	".example":  {},
	".template": {},
	".sample":   {},
	".dist":     {},
	".default":  {},
	".tmpl":     {},
	".tpl":      {},
}

// SupportedExtensions maps file extensions (and exact filenames) to language identifiers.
var SupportedExtensions = map[string]string{
	// Go
	".go":  "go",
	".mod": "go",
	".sum": "text",

	// TypeScript / JavaScript
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "javascript",
	".jsx": "javascript",
	".mjs": "javascript",
	".cjs": "javascript",

	// Python
	".py":  "python",
	".pyw": "python",

	// Web standards
	".html": "html",
	".htm":  "html",
	".css":  "css",
	".scss": "scss",
	".less": "less",
	".svg":  "xml",

	// Data formats & config
	".json":  "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".toml":  "toml",
	".xml":   "xml",
	".csv":   "csv",
	".ini":   "ini",
	".cfg":   "ini",
	".env":   "dotenv",
	".proto": "protobuf",

	// Terraform / HCL
	".tf":     "terraform",
	".tfvars": "terraform",
	".tftpl":  "terraform",
	".hcl":    "hcl",

	// Systems / Languages
	".c":     "c",
	".h":     "cpp",
	".cpp":   "cpp",
	".hpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".rs":    "rust",
	".java":  "java",
	".cs":    "csharp",
	".swift": "swift",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".scala": "scala",
	".rb":    "ruby",
	".php":   "php",
	".sql":   "sql",
	".sh":    "bash",
	".bash":  "bash",
	".zsh":   "bash",
	".ps1":   "powershell",
	".r":     "r",
	".dart":  "dart",
	".lua":   "lua",

	// Documentation / Text
	".md":       "markdown",
	".txt":      "text",
	".rst":      "rst",
	".pdf":      "pdf",
	".log":      "text",
	".make":     "makefile",
	".docker":   "dockerfile",
	".gitattributes": "gitattributes",
	".gitignore":     "gitignore",
}

// SpecialFilenames maps exact filenames (case-insensitive) to language identifiers.
var SpecialFilenames = map[string]string{
	"makefile":   "makefile",
	"dockerfile": "dockerfile",
	"cmakelists.txt": "cmake",
	"license":    "text",
	"readme":     "markdown",
}

// StripAuxiliaryExtensions iteratively removes trailing auxiliary extensions from a path or filename.
func StripAuxiliaryExtensions(path string) string {
	for {
		originalExt := filepath.Ext(path)
		ext := strings.ToLower(originalExt)
		if _, ok := AuxiliaryExtensions[ext]; !ok {
			break
		}
		path = strings.TrimSuffix(path, originalExt)
	}
	return path
}

// LookupLanguage resolves a file path to a language identifier and indicates whether the extension/filename is in the supported list.
func LookupLanguage(path string) (string, bool) {
	stripped := StripAuxiliaryExtensions(path)

	base := strings.ToLower(filepath.Base(stripped))
	if lang, ok := SpecialFilenames[base]; ok {
		return lang, true
	}

	ext := strings.ToLower(filepath.Ext(stripped))
	if ext == "" {
		return "text", false
	}

	if lang, ok := SupportedExtensions[ext]; ok {
		return lang, true
	}

	// Unknown extension: return extension without dot (or "text") and false
	cleanExt := strings.TrimPrefix(ext, ".")
	if cleanExt == "" {
		cleanExt = "text"
	}
	return cleanExt, false
}

// FormatEntry represents a file extension and its mapped language identifier.
type FormatEntry struct {
	Extension string
	Language  string
}

// SpecialFileEntry represents a special filename and its mapped language identifier.
type SpecialFileEntry struct {
	Filename string
	Language string
}

// ListFormats returns sorted slices of supported file extensions and special filenames.
func ListFormats() ([]FormatEntry, []SpecialFileEntry, error) {
	exts := make([]FormatEntry, 0, len(SupportedExtensions))
	for ext, lang := range SupportedExtensions {
		exts = append(exts, FormatEntry{Extension: ext, Language: lang})
	}
	slices.SortFunc(exts, func(a, b FormatEntry) int {
		return cmp.Compare(a.Extension, b.Extension)
	})

	specials := make([]SpecialFileEntry, 0, len(SpecialFilenames))
	for file, lang := range SpecialFilenames {
		specials = append(specials, SpecialFileEntry{Filename: file, Language: lang})
	}
	slices.SortFunc(specials, func(a, b SpecialFileEntry) int {
		return cmp.Compare(a.Filename, b.Filename)
	})

	return exts, specials, nil
}

