package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LoginScreen struct {
	Container *fyne.Container
	OnConnect func(host, port, user, pass, key, logPath string)
}

func NewLoginScreen(onConnect func(host, port, user, pass, key, logPath string)) *LoginScreen {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("Host (e.g., 192.168.1.10)")
	hostEntry.Text = "localhost" // Default for testing

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Port")
	portEntry.Text = "22"

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("Username")

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Password")

	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("Private Key Path (Optional)")

	logPathEntry := widget.NewEntry()
	logPathEntry.SetPlaceHolder("Remote Log Path (e.g., /var/log/logstash)")
	logPathEntry.Text = "/var/log/logstash"

	// Custom layout to center the button
	connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		if onConnect != nil {
			onConnect(
				hostEntry.Text,
				portEntry.Text,
				userEntry.Text,
				passEntry.Text,
				keyEntry.Text,
				logPathEntry.Text,
			)
		}
	})
	connectBtn.Importance = widget.HighImportance

	formContent := container.NewVBox(
		widget.NewLabel("Host"), hostEntry,
		widget.NewLabel("Port"), portEntry,
		widget.NewLabel("Username"), userEntry,
		widget.NewLabel("Password"), passEntry,
		widget.NewLabel("Key Path"), keyEntry,
		widget.NewLabel("Log Path"), logPathEntry,
		widget.NewLabel(""), // Spacer
		container.NewCenter(connectBtn),
	)

	// Create a card to contain the login form for better visibility and grouping
	card := widget.NewCard("Connection Details", "", formContent)

	// Wrap in a container that provides some minimum width
	// We use a GridWrapLayout to force a minimum size for the card
	sizedContent := container.NewGridWrap(fyne.NewSize(400, 400), card)

	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Logstash SSH Client", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			sizedContent,
		),
	)

	return &LoginScreen{
		Container: content,
		OnConnect: onConnect,
	}
}
