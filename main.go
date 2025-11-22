package main

import (
	"logsearch/ssh"
	"logsearch/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	a := app.New()
	w := a.NewWindow("Logstash SSH Client")
	w.Resize(fyne.NewSize(1024, 768))

	var client *ssh.Client

	// Define the transition to dashboard
	showDashboard := func(host, port, user, pass, key, logPath string) {
		client = ssh.NewClient()
		err := client.Connect(host, port, user, pass, key)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		dashboard := ui.NewDashboard(w, client, logPath)
		w.SetContent(dashboard.Container)
	}

	// Initial screen is Login
	loginScreen := ui.NewLoginScreen(showDashboard)
	w.SetContent(loginScreen.Container)

	w.ShowAndRun()

	if client != nil {
		client.Close()
	}
}
