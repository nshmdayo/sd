package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for zcd
type Config struct {
	DataFile     string `json:"data_file"`
	MaxHistory   int    `json:"max_history"`
	MaxBookmarks int    `json:"max_bookmarks"`
}

// DirectoryEntry represents a directory entry with metadata
type DirectoryEntry struct {
	Path       string    `json:"path"`
	Count      int       `json:"count"`
	LastUsed   time.Time `json:"last_used"`
	IsBookmark bool      `json:"is_bookmark"`
}

// ZCD represents the main application state
type ZCD struct {
	config      Config
	directories map[string]*DirectoryEntry
	dataFile    string
}

// NewZCD creates a new ZCD instance
func NewZCD() *ZCD {
	homeDir, _ := os.UserHomeDir()
	dataFile := filepath.Join(homeDir, ".zcd_data.json")

	config := Config{
		DataFile:     dataFile,
		MaxHistory:   100,
		MaxBookmarks: 50,
	}

	zcd := &ZCD{
		config:      config,
		directories: make(map[string]*DirectoryEntry),
		dataFile:    dataFile,
	}

	zcd.loadData()
	return zcd
}

// loadData loads directory data from the JSON file
func (z *ZCD) loadData() {
	if _, err := os.Stat(z.dataFile); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(z.dataFile)
	if err != nil {
		return
	}

	var entries []DirectoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	for _, entry := range entries {
		z.directories[entry.Path] = &entry
	}
}

// saveData saves directory data to the JSON file
func (z *ZCD) saveData() error {
	var entries []DirectoryEntry
	for _, entry := range z.directories {
		entries = append(entries, *entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(z.dataFile, data, 0644)
}

// addDirectory adds or updates a directory entry
func (z *ZCD) addDirectory(path string, isBookmark bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if entry, exists := z.directories[absPath]; exists {
		entry.Count++
		entry.LastUsed = time.Now()
		if isBookmark {
			entry.IsBookmark = true
		}
	} else {
		z.directories[absPath] = &DirectoryEntry{
			Path:       absPath,
			Count:      1,
			LastUsed:   time.Now(),
			IsBookmark: isBookmark,
		}
	}

	// Cleanup old entries if we exceed max history
	z.cleanupOldEntries()
}

// cleanupOldEntries removes old non-bookmark entries
func (z *ZCD) cleanupOldEntries() {
	var nonBookmarks []*DirectoryEntry
	for _, entry := range z.directories {
		if !entry.IsBookmark {
			nonBookmarks = append(nonBookmarks, entry)
		}
	}

	if len(nonBookmarks) > z.config.MaxHistory {
		// Sort by last used time (oldest first)
		sort.Slice(nonBookmarks, func(i, j int) bool {
			return nonBookmarks[i].LastUsed.Before(nonBookmarks[j].LastUsed)
		})

		// Remove oldest entries
		toRemove := len(nonBookmarks) - z.config.MaxHistory
		for i := 0; i < toRemove; i++ {
			delete(z.directories, nonBookmarks[i].Path)
		}
	}
}

// getSuggestions returns directory suggestions based on input
func (z *ZCD) getSuggestions(input string) []DirectoryEntry {
	var suggestions []DirectoryEntry

	for _, entry := range z.directories {
		// Check if the directory still exists
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			continue
		}

		// Match by basename or full path
		baseName := filepath.Base(entry.Path)
		if strings.Contains(strings.ToLower(baseName), strings.ToLower(input)) ||
			strings.Contains(strings.ToLower(entry.Path), strings.ToLower(input)) {
			suggestions = append(suggestions, *entry)
		}
	}

	// Sort suggestions by score (bookmarks first, then by count and recency)
	sort.Slice(suggestions, func(i, j int) bool {
		a, b := suggestions[i], suggestions[j]

		// Bookmarks have priority
		if a.IsBookmark && !b.IsBookmark {
			return true
		}
		if !a.IsBookmark && b.IsBookmark {
			return false
		}

		// Then by count
		if a.Count != b.Count {
			return a.Count > b.Count
		}

		// Then by recency
		return a.LastUsed.After(b.LastUsed)
	})

	// Limit to top 10 suggestions
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions
}

// listHistory shows the directory history
func (z *ZCD) listHistory() {
	var entries []DirectoryEntry
	for _, entry := range z.directories {
		// Check if the directory still exists
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			continue
		}
		entries = append(entries, *entry)
	}

	// Sort by last used (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsed.After(entries[j].LastUsed)
	})

	fmt.Println("Directory History:")
	fmt.Println("==================")
	for i, entry := range entries {
		if i >= 20 { // Show only last 20 entries
			break
		}

		bookmark := ""
		if entry.IsBookmark {
			bookmark = " [★]"
		}

		fmt.Printf("%2d. %s (used %d times)%s\n", i+1, entry.Path, entry.Count, bookmark)
	}
}

// listBookmarks shows the bookmarked directories
func (z *ZCD) listBookmarks() {
	var bookmarks []DirectoryEntry
	for _, entry := range z.directories {
		if entry.IsBookmark {
			// Check if the directory still exists
			if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
				continue
			}
			bookmarks = append(bookmarks, *entry)
		}
	}

	// Sort by path
	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].Path < bookmarks[j].Path
	})

	fmt.Println("Bookmarked Directories:")
	fmt.Println("======================")
	for i, entry := range bookmarks {
		fmt.Printf("%2d. %s (used %d times)\n", i+1, entry.Path, entry.Count)
	}
}

// changeDirectory handles the cd operation
func (z *ZCD) changeDirectory(target string) error {
	var targetPath string

	if target == "" {
		// No argument, go to home directory
		homeDir, _ := os.UserHomeDir()
		targetPath = homeDir
	} else if target == "-" {
		// Go to previous directory (OLDPWD)
		targetPath = os.Getenv("OLDPWD")
		if targetPath == "" {
			return fmt.Errorf("OLDPWD not set")
		}
	} else if num, err := strconv.Atoi(target); err == nil {
		// Numeric input, select from history
		var entries []DirectoryEntry
		for _, entry := range z.directories {
			if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
				continue
			}
			entries = append(entries, *entry)
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].LastUsed.After(entries[j].LastUsed)
		})

		if num < 1 || num > len(entries) {
			return fmt.Errorf("invalid history number: %d", num)
		}

		targetPath = entries[num-1].Path
	} else {
		// Regular path or suggestion search
		if filepath.IsAbs(target) {
			targetPath = target
		} else {
			// Check if it's a relative path
			if _, err := os.Stat(target); err == nil {
				abs, _ := filepath.Abs(target)
				targetPath = abs
			} else {
				// Search for suggestions
				suggestions := z.getSuggestions(target)
				if len(suggestions) == 0 {
					return fmt.Errorf("no matching directories found for: %s", target)
				}

				if len(suggestions) == 1 {
					targetPath = suggestions[0].Path
				} else {
					// Multiple suggestions, show them
					fmt.Println("Multiple matches found:")
					for i, suggestion := range suggestions {
						bookmark := ""
						if suggestion.IsBookmark {
							bookmark = " [★]"
						}
						fmt.Printf("%2d. %s%s\n", i+1, suggestion.Path, bookmark)
					}

					fmt.Print("Select number (1-" + strconv.Itoa(len(suggestions)) + "): ")
					reader := bufio.NewReader(os.Stdin)
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(input)

					if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(suggestions) {
						targetPath = suggestions[num-1].Path
					} else {
						return fmt.Errorf("invalid selection")
					}
				}
			}
		}
	}

	// Verify the target directory exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", targetPath)
	}

	// Record the directory visit
	z.addDirectory(targetPath, false)

	// Save current directory as OLDPWD
	currentDir, _ := os.Getwd()
	os.Setenv("OLDPWD", currentDir)

	// Change to the target directory
	if err := os.Chdir(targetPath); err != nil {
		return err
	}

	// Save data
	z.saveData()

	fmt.Printf("Changed to: %s\n", targetPath)
	return nil
}

// bookmark adds a directory to bookmarks
func (z *ZCD) bookmark(path string) error {
	if path == "" {
		// Bookmark current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}
		path = currentDir
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Verify the directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absPath)
	}

	z.addDirectory(absPath, true)
	z.saveData()

	fmt.Printf("Bookmarked: %s\n", absPath)
	return nil
}

// removeBookmark removes a directory from bookmarks
func (z *ZCD) removeBookmark(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if entry, exists := z.directories[absPath]; exists {
		entry.IsBookmark = false
		z.saveData()
		fmt.Printf("Removed bookmark: %s\n", absPath)
	} else {
		return fmt.Errorf("bookmark not found: %s", absPath)
	}

	return nil
}

// showUsage displays the usage information
func showUsage() {
	fmt.Println("zcd - Smart directory navigation")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  zcd [directory]       Change to directory (with smart suggestions)")
	fmt.Println("  zcd                   Change to home directory")
	fmt.Println("  zcd -                 Change to previous directory")
	fmt.Println("  zcd [number]          Change to directory from history")
	fmt.Println("  zcd -h, --history     Show directory history")
	fmt.Println("  zcd -b, --bookmarks   Show bookmarked directories")
	fmt.Println("  zcd -a, --add [path]  Add bookmark (current dir if no path)")
	fmt.Println("  zcd -r, --remove path Remove bookmark")
	fmt.Println("  zcd --help            Show this help")
}

func main() {
	zcd := NewZCD()

	args := os.Args[1:]

	if len(args) == 0 {
		// No arguments, change to home directory
		if err := zcd.changeDirectory(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "--help":
		showUsage()
	case "-h", "--history":
		zcd.listHistory()
	case "-b", "--bookmarks":
		zcd.listBookmarks()
	case "-a", "--add":
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		if err := zcd.bookmark(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "-r", "--remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: path required for remove command\n")
			os.Exit(1)
		}
		if err := zcd.removeBookmark(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		// Regular directory change
		target := strings.Join(args, " ")
		if err := zcd.changeDirectory(target); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
