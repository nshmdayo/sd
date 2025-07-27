package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewZCD(t *testing.T) {
	zcd := NewZCD()
	if zcd == nil {
		t.Fatal("NewZCD() returned nil")
	}

	if zcd.directories == nil {
		t.Fatal("directories map not initialized")
	}

	if zcd.config.MaxHistory != 100 {
		t.Errorf("Expected MaxHistory to be 100, got %d", zcd.config.MaxHistory)
	}
}

func TestAddDirectory(t *testing.T) {
	zcd := NewZCD()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Add directory
	zcd.addDirectory(tempDir, false)

	// Check if directory was added
	if entry, exists := zcd.directories[tempDir]; !exists {
		t.Fatal("Directory was not added")
	} else {
		if entry.Count != 1 {
			t.Errorf("Expected count to be 1, got %d", entry.Count)
		}
		if entry.IsBookmark {
			t.Error("Expected IsBookmark to be false")
		}
	}

	// Add same directory again
	zcd.addDirectory(tempDir, false)

	if entry := zcd.directories[tempDir]; entry.Count != 2 {
		t.Errorf("Expected count to be 2, got %d", entry.Count)
	}
}

func TestAddBookmark(t *testing.T) {
	zcd := NewZCD()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Add as bookmark
	zcd.addDirectory(tempDir, true)

	// Check if directory was added as bookmark
	if entry, exists := zcd.directories[tempDir]; !exists {
		t.Fatal("Bookmark was not added")
	} else {
		if !entry.IsBookmark {
			t.Error("Expected IsBookmark to be true")
		}
	}
}

func TestGetSuggestions(t *testing.T) {
	zcd := NewZCD()

	// Create temporary directories for testing
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Add directories with known names
	testDir1 := filepath.Join(tempDir1, "project1")
	testDir2 := filepath.Join(tempDir2, "project2")

	os.Mkdir(testDir1, 0755)
	os.Mkdir(testDir2, 0755)

	zcd.addDirectory(testDir1, false)
	zcd.addDirectory(testDir2, true) // This one as bookmark

	// Test suggestions
	suggestions := zcd.getSuggestions("project")

	if len(suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(suggestions))
	}

	// Bookmark should come first
	if len(suggestions) > 0 && !suggestions[0].IsBookmark {
		t.Error("Expected bookmark to be first in suggestions")
	}
}

func TestCleanupOldEntries(t *testing.T) {
	zcd := NewZCD()
	zcd.config.MaxHistory = 2 // Set low limit for testing

	// Create temporary directories
	tempDirs := make([]string, 3)
	for i := 0; i < 3; i++ {
		tempDirs[i] = t.TempDir()
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		zcd.addDirectory(tempDirs[i], false)
	}

	// Should have only 2 entries (oldest removed)
	nonBookmarkCount := 0
	for _, entry := range zcd.directories {
		if !entry.IsBookmark {
			nonBookmarkCount++
		}
	}

	if nonBookmarkCount > zcd.config.MaxHistory {
		t.Errorf("Expected at most %d non-bookmark entries, got %d", zcd.config.MaxHistory, nonBookmarkCount)
	}
}

func TestBookmarkOperations(t *testing.T) {
	zcd := NewZCD()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test bookmark function
	err := zcd.bookmark(tempDir)
	if err != nil {
		t.Fatalf("Failed to add bookmark: %v", err)
	}

	// Check if it was bookmarked
	if entry, exists := zcd.directories[tempDir]; !exists {
		t.Fatal("Bookmark was not added")
	} else if !entry.IsBookmark {
		t.Error("Directory was not marked as bookmark")
	}

	// Test remove bookmark
	err = zcd.removeBookmark(tempDir)
	if err != nil {
		t.Fatalf("Failed to remove bookmark: %v", err)
	}

	// Check if bookmark was removed
	if entry := zcd.directories[tempDir]; entry.IsBookmark {
		t.Error("Bookmark was not removed")
	}
}

func TestSaveAndLoadData(t *testing.T) {
	// Create a temporary file for testing
	tempFile := filepath.Join(t.TempDir(), "test_data.json")

	zcd := &ZCD{
		config: Config{
			DataFile:     tempFile,
			MaxHistory:   100,
			MaxBookmarks: 50,
		},
		directories: make(map[string]*DirectoryEntry),
		dataFile:    tempFile,
	}

	// Add some test data
	testDir := t.TempDir()
	zcd.addDirectory(testDir, true)

	// Save data
	err := zcd.saveData()
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	// Create new instance and load data
	zcd2 := &ZCD{
		config: Config{
			DataFile:     tempFile,
			MaxHistory:   100,
			MaxBookmarks: 50,
		},
		directories: make(map[string]*DirectoryEntry),
		dataFile:    tempFile,
	}

	zcd2.loadData()

	// Check if data was loaded correctly
	if entry, exists := zcd2.directories[testDir]; !exists {
		t.Fatal("Data was not loaded correctly")
	} else if !entry.IsBookmark {
		t.Error("Bookmark status was not preserved")
	}
}
