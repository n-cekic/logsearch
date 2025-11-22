package ui

import (
	"fmt"
	"logsearch/ssh"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Dashboard struct {
	Container   *fyne.Container
	sshClient   *ssh.Client
	currentPath string
	window      fyne.Window

	fileList    *widget.List
	fileEntries []ssh.FileEntry
	selected    map[int]bool // Track selected indices
	contentArea *widget.Entry
	pathLabel   *widget.Label

	searchEntry *widget.Entry
}

func NewDashboard(window fyne.Window, client *ssh.Client, initialPath string) *Dashboard {
	d := &Dashboard{
		sshClient:   client,
		currentPath: initialPath,
		window:      window,
		selected:    make(map[int]bool),
	}

	d.pathLabel = widget.NewLabel(initialPath)
	d.pathLabel.TextStyle = fyne.TextStyle{Monospace: true}

	d.fileList = widget.NewList(
		func() int {
			return len(d.fileEntries)
		},
		func() fyne.CanvasObject {
			// Checkbox + Icon + Label
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			entry := d.fileEntries[id]
			box := item.(*fyne.Container)
			check := box.Objects[0].(*widget.Check)
			icon := box.Objects[1].(*widget.Icon)
			label := box.Objects[2].(*widget.Label)

			label.SetText(entry.Name)
			if entry.IsDir {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
			}

			check.OnChanged = nil // Avoid triggering during update
			check.SetChecked(d.selected[id])
			check.OnChanged = func(checked bool) {
				if checked {
					d.selected[id] = true
				} else {
					delete(d.selected, id)
				}
			}
		},
	)

	d.fileList.OnSelected = func(id widget.ListItemID) {
		entry := d.fileEntries[id]
		fullPath := filepath.Join(d.currentPath, entry.Name)

		if entry.IsDir {
			d.refreshFileList(fullPath)
			d.fileList.UnselectAll()
		} else {
			d.loadFileContent(fullPath)
		}
	}

	d.contentArea = widget.NewMultiLineEntry()
	d.contentArea.TextStyle = fyne.TextStyle{Monospace: true}
	d.contentArea.Wrapping = fyne.TextWrapOff

	// Back button
	backBtn := widget.NewButtonWithIcon("Up", theme.NavigateBackIcon(), func() {
		parent := filepath.Dir(d.currentPath)
		if parent != "." && parent != "/" {
			d.refreshFileList(parent)
		} else if parent == "/" {
			d.refreshFileList("/")
		}
	})

	// Search UI
	d.searchEntry = widget.NewEntry()
	d.searchEntry.SetPlaceHolder("Regex Pattern")
	searchBtn := widget.NewButtonWithIcon("Search", theme.SearchIcon(), d.performSearch)

	leftPane := container.NewBorder(
		container.NewVBox(widget.NewLabel("Files"), backBtn),
		container.NewVBox(d.searchEntry, searchBtn),
		nil, nil,
		d.fileList,
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

	// Initial load
	d.refreshFileList(initialPath)

	return d
}

func (d *Dashboard) refreshFileList(path string) {
	entries, err := d.sshClient.ListDir(path)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

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

	d.fileEntries = entries
	d.selected = make(map[int]bool) // Reset selection on nav
	d.currentPath = path
	d.pathLabel.SetText(path)
	d.fileList.Refresh()
}

func (d *Dashboard) loadFileContent(path string) {
	d.contentArea.SetText("Loading...")

	// Run in goroutine to avoid freezing UI
	go func() {
		content, err := d.sshClient.ReadFile(path)
		if err != nil {
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
	for id := range d.selected {
		if id < len(d.fileEntries) {
			paths = append(paths, filepath.Join(d.currentPath, d.fileEntries[id].Name))
		}
	}

	d.contentArea.SetText("Searching...")

	go func() {
		results, err := d.sshClient.Search(paths, pattern)
		if err != nil {
			d.contentArea.SetText(fmt.Sprintf("Search error: %v", err))
			return
		}

		if results == "" {
			d.contentArea.SetText("No matches found.")
		} else {
			d.contentArea.SetText(results)
		}
	}()
}
