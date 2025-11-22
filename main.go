package main

import (
	"log"
	"logsearch/ssh"
	"logsearch/ui"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	// Setup logging
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	log.Println("Application started")

	a := app.New()
	w := a.NewWindow("Logstash SSH Client")
	w.Resize(fyne.NewSize(1024, 768))

	var client *ssh.Client

	// Define the transition to dashboard
	showDashboard := func(host, port, user, pass, key, logPath string) {
		client = ssh.NewClient()
		err := client.Connect(host, port, user, pass, key)
		if err != nil {
			log.Printf("Connection failed: %v", err)
			dialog.ShowError(err, w)
			return
		}
		log.Printf("Connected to %s:%s as %s", host, port, user)

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
	log.Println("Application stopped")
}
