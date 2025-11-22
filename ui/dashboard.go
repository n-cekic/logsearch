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
	contentArea *widget.Entry
	pathLabel   *widget.Label
}

func NewDashboard(window fyne.Window, client *ssh.Client, initialPath string) *Dashboard {
	d := &Dashboard{
		sshClient:   client,
		currentPath: initialPath,
		window:      window,
	}

	d.pathLabel = widget.NewLabel(initialPath)
	d.pathLabel.TextStyle = fyne.TextStyle{Monospace: true}

	d.fileList = widget.NewList(
		func() int {
			return len(d.fileEntries)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.FileIcon()), widget.NewLabel("Template"))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			entry := d.fileEntries[id]
			box := item.(*fyne.Container)
			icon := box.Objects[0].(*widget.Icon)
			label := box.Objects[1].(*widget.Label)

			label.SetText(entry.Name)
			if entry.IsDir {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
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
	d.contentArea.Wrapping = fyne.TextWrapOff // Better for logs

	// Back button
	backBtn := widget.NewButtonWithIcon("Up", theme.NavigateBackIcon(), func() {
		parent := filepath.Dir(d.currentPath)
		if parent != "." && parent != "/" {
			d.refreshFileList(parent)
		} else if parent == "/" {
			d.refreshFileList("/")
		}
	})

	leftPane := container.NewBorder(
		container.NewVBox(widget.NewLabel("Files"), backBtn),
		nil, nil, nil,
		d.fileList,
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
