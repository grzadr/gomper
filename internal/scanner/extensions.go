package scanner

import (
	"path/filepath"
	"strings"
)

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

// LookupLanguage resolves a file path to a language identifier and indicates whether the extension/filename is in the supported list.
func LookupLanguage(path string) (string, bool) {
	base := strings.ToLower(filepath.Base(path))
	if lang, ok := SpecialFilenames[base]; ok {
		return lang, true
	}

	ext := strings.ToLower(filepath.Ext(path))
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
