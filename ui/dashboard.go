package ui

import (
	"fmt"
	"log"
	"logsearch/ssh"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Dashboard struct {
	Container   *fyne.Container
	sshClient   *ssh.Client
	currentPath string
	rootPath    string
	window      fyne.Window

	fileTree       *widget.Tree
	selected       map[string]bool // Track selected paths
	contentArea    *widget.Entry
	contentBinding binding.String
	pathLabel      *widget.Label

	searchEntry *widget.Entry

	// Cache for directory contents
	dirCache map[string][]ssh.FileEntry
}

func NewDashboard(window fyne.Window, client *ssh.Client, initialPath string) *Dashboard {
	d := &Dashboard{
		sshClient:      client,
		currentPath:    initialPath,
		rootPath:       initialPath,
		window:         window,
		selected:       make(map[string]bool),
		contentBinding: binding.NewString(),
		dirCache:       make(map[string][]ssh.FileEntry),
	}

	d.pathLabel = widget.NewLabel(initialPath)
	d.pathLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// Create tree widget
	d.fileTree = widget.NewTree(
		// ChildUIDs: Return child node IDs for a given parent
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return d.getChildUIDs(uid)
		},
		// IsBranch: Determine if a node is a directory
		func(uid widget.TreeNodeID) bool {
			return d.isBranch(uid)
		},
		// CreateNode: Create the visual template for a node
		func(branch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("Template"),
			)
		},
		// UpdateNode: Update the node with actual data
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			d.updateNode(uid, branch, obj)
		},
	)

	// Handle node selection (clicking the label/icon, not checkbox)
	d.fileTree.OnSelected = func(uid widget.TreeNodeID) {
		path := string(uid)
		if d.isBranch(uid) {
			// Directory selected - toggle expansion
			if d.fileTree.IsBranchOpen(uid) {
				d.fileTree.CloseBranch(uid)
			} else {
				d.fileTree.OpenBranch(uid)
			}
			// Keep selection to show current location, or unselect if preferred
			// d.fileTree.Unselect(uid)
		} else {
			// File selected - load content
			d.loadFileContent(path)
		}
		d.pathLabel.SetText(path)
	}

	d.contentArea = widget.NewEntryWithData(d.contentBinding)
	d.contentArea.MultiLine = true
	d.contentArea.TextStyle = fyne.TextStyle{Monospace: true}
	d.contentArea.Wrapping = fyne.TextWrapOff

	// Search UI
	d.searchEntry = widget.NewEntry()
	d.searchEntry.SetPlaceHolder("Regex Pattern")
	searchBtn := widget.NewButtonWithIcon("Search", theme.SearchIcon(), d.performSearch)

	leftPane := container.NewBorder(
		widget.NewLabel("Files"),
		container.NewVBox(d.searchEntry, searchBtn),
		nil, nil,
		d.fileTree,
	)

	rightPane := container.NewBorder(
		widget.NewLabel("Content / Results"),
		nil, nil, nil,
		d.contentArea,
	)

	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.3)

	d.Container = container.NewBorder(
		container.NewVBox(widget.NewLabel("Location:"), d.pathLabel),
		nil, nil, nil,
		split,
	)

	// Initial load - open the root path
	d.fileTree.OpenBranch(widget.TreeNodeID(initialPath))

	return d
}

// getChildUIDs returns the child node IDs for a given parent
func (d *Dashboard) getChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	var path string
	if uid == "" {
		// Root level - return the initial path
		path = d.rootPath
		return []widget.TreeNodeID{widget.TreeNodeID(path)}
	}

	path = string(uid)

	// Check cache first
	if entries, ok := d.dirCache[path]; ok {
		log.Printf("Using cached entries for %s", path)
		return d.entriesToUIDs(path, entries)
	}

	// Load directory contents
	log.Printf("Listing directory: %s", path)
	entries, err := d.sshClient.ListDir(path)
	if err != nil {
		log.Printf("Failed to list directory %s: %v", path, err)
		return []widget.TreeNodeID{}
	}
	log.Printf("Found %d entries in %s", len(entries), path)

	// Sort: Directories first, then files
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir && !entries[j].IsDir {
			return true
		}
		if !entries[i].IsDir && entries[j].IsDir {
			return false
		}
		return entries[i].Name < entries[j].Name
	})

	// Cache the results
	d.dirCache[path] = entries

	return d.entriesToUIDs(path, entries)
}

// entriesToUIDs converts file entries to tree node IDs
func (d *Dashboard) entriesToUIDs(parentPath string, entries []ssh.FileEntry) []widget.TreeNodeID {
	var uids []widget.TreeNodeID
	for _, entry := range entries {
		fullPath := filepath.Join(parentPath, entry.Name)
		uids = append(uids, widget.TreeNodeID(fullPath))
	}
	return uids
}

// isBranch determines if a node is a directory
func (d *Dashboard) isBranch(uid widget.TreeNodeID) bool {
	if uid == "" {
		return true // Root is always a branch
	}

	path := string(uid)

	// Check if this path is in our cache
	parentPath := filepath.Dir(path)
	if entries, ok := d.dirCache[parentPath]; ok {
		name := filepath.Base(path)
		for _, entry := range entries {
			if entry.Name == name {
				return entry.IsDir
			}
		}
	}

	// If not in cache (e.g. root), treat as branch if it's the root path
	if path == d.rootPath {
		return true
	}

	// Fallback: try to list it
	_, err := d.sshClient.ListDir(path)
	return err == nil
}

// updateNode updates the visual representation of a node
func (d *Dashboard) updateNode(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
	path := string(uid)
	name := filepath.Base(path)

	// Handle root node specially
	if strings.Count(path, "/") <= 1 && path != "" {
		name = path
	}

	box := obj.(*fyne.Container)
	check := box.Objects[0].(*widget.Check)
	icon := box.Objects[1].(*widget.Icon)
	label := box.Objects[2].(*widget.Label)

	label.SetText(name)
	if branch {
		icon.SetResource(theme.FolderIcon())
	} else {
		icon.SetResource(theme.FileIcon())
	}

	check.OnChanged = nil // Avoid triggering during update
	check.SetChecked(d.selected[path])
	check.OnChanged = func(checked bool) {
		if checked {
			d.selected[path] = true
		} else {
			delete(d.selected, path)
		}
	}
}

func (d *Dashboard) loadFileContent(path string) {
	log.Printf("Loading file content: %s", path)
	d.contentBinding.Set("Loading...")

	// Run in goroutine to avoid freezing UI
	go func() {
		content, err := d.sshClient.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			d.contentBinding.Set(fmt.Sprintf("Error reading file: %v", err))
			return
		}

		log.Printf("Read %d bytes from %s", len(content), path)

		// Limit content size for performance
		text := string(content)
		if len(text) > 100000 {
			log.Printf("Truncating file content for display (original size: %d)", len(text))
			text = text[len(text)-100000:] // Show last 100KB
			text = "[Truncated... showing last 100KB]\n" + text
		}

		d.contentBinding.Set(text)
	}()
}

func (d *Dashboard) performSearch() {
	if len(d.selected) == 0 {
		dialog.ShowInformation("Search", "Please select at least one file or folder.", d.window)
		return
	}

	pattern := d.searchEntry.Text
	if pattern == "" {
		dialog.ShowInformation("Search", "Please enter a search pattern.", d.window)
		return
	}

	var paths []string
	for path := range d.selected {
		paths = append(paths, path)
	}

	d.contentBinding.Set("Searching...")

	go func() {
		results, err := d.sshClient.Search(paths, pattern)
		if err != nil {
			d.contentBinding.Set(fmt.Sprintf("Search error: %v", err))
			return
		}

		if results == "" {
			d.contentBinding.Set("No matches found.")
		} else {
			d.contentBinding.Set(results)
		}
	}()
}
