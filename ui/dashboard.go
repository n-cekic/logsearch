package ui

import (
	"fmt"
	"log"
	"logsearch/ssh"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Dashboard struct {
	Container   *fyne.Container
	sshClient   *ssh.Client
	currentPath string
	window      fyne.Window

	fileTree    *widget.Tree
	contentArea *widget.Entry
	pathLabel   *widget.Label

	// Cache for directory contents
	dirCache map[string][]ssh.FileEntry
}

func NewDashboard(window fyne.Window, client *ssh.Client, initialPath string) *Dashboard {
	d := &Dashboard{
		sshClient:   client,
		currentPath: initialPath,
		window:      window,
		dirCache:    make(map[string][]ssh.FileEntry),
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
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("Template"),
			)
		},
		// UpdateNode: Update the node with actual data
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			d.updateNode(uid, branch, obj)
		},
	)

	// Handle node selection
	d.fileTree.OnSelected = func(uid widget.TreeNodeID) {
		path := string(uid)
		if d.isBranch(uid) {
			// Directory selected - update path label
			d.pathLabel.SetText(path)
		} else {
			// File selected - load content
			d.loadFileContent(path)
			d.pathLabel.SetText(path)
		}
	}

	d.contentArea = widget.NewMultiLineEntry()
	d.contentArea.TextStyle = fyne.TextStyle{Monospace: true}
	d.contentArea.Wrapping = fyne.TextWrapOff // Better for logs

	leftPane := container.NewBorder(
		widget.NewLabel("Files"),
		nil, nil, nil,
		d.fileTree,
	)

	rightPane := container.NewBorder(
		widget.NewLabel("Content"),
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
		path = d.currentPath
		return []widget.TreeNodeID{widget.TreeNodeID(path)}
	}

	path = string(uid)

	// Check cache first
	if entries, ok := d.dirCache[path]; ok {
		return d.entriesToUIDs(path, entries)
	}

	// Load directory contents
	entries, err := d.sshClient.ListDir(path)
	if err != nil {
		log.Printf("Failed to list directory %s: %v", path, err)
		return []widget.TreeNodeID{}
	}

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

	// If not in cache, try to list it to see if it's a directory
	// This is a fallback and shouldn't happen often
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
	icon := box.Objects[0].(*widget.Icon)
	label := box.Objects[1].(*widget.Label)

	label.SetText(name)
	if branch {
		icon.SetResource(theme.FolderIcon())
	} else {
		icon.SetResource(theme.FileIcon())
	}
}

// loadFileContent loads and displays file content
func (d *Dashboard) loadFileContent(path string) {
	d.contentArea.SetText("Loading...")

	// Run in goroutine to avoid freezing UI
	go func() {
		content, err := d.sshClient.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			d.contentArea.SetText(fmt.Sprintf("Error reading file: %v", err))
			return
		}

		// Limit content size for performance
		text := string(content)
		if len(text) > 100000 {
			text = text[len(text)-100000:] // Show last 100KB
			text = "[Truncated... showing last 100KB]\n" + text
		}

		d.contentArea.SetText(text)
	}()
}
