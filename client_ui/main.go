package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	appName            = "jaaTorrent Client GUI"
	appID              = "jaatorrent-client-gui"
	customBackLabel    = "NADA"
	defaultInputFormat = "Generic"
)

func showErrorf(win fyne.Window, format string, args ...interface{}) {
	msg := fmt.Errorf(format, args...)
	//log.Info(msg)
	dialog.ShowError(msg, win)
}

func newProgressBar(title string, win fyne.Window) dialog.Dialog {
	bar := widget.NewProgressBarInfinite()
	bar.Resize(fyne.NewSize(200, bar.MinSize().Height))

	progress := dialog.NewCustom(title, "Cancel", bar, win)

	return progress
}

func showAboutWindow(app fyne.App) {
	aboutWindow := app.NewWindow("About")

	var aboutMsg strings.Builder

	aboutMsg.WriteString(appName)
	aboutMsg.WriteString("\n\nBuilt with Go")
	aboutMsg.WriteString(" and Fyne v2 UI")

	aboutLabel := widget.NewLabel(aboutMsg.String())

	okButton := widget.NewButton("OK", func() {
		aboutWindow.Close()
	})

	buttons := container.New(layout.NewHBoxLayout(),
		layout.NewSpacer(),
		okButton,
		layout.NewSpacer(),
	)

	repoURL, _ := url.Parse("https://github.com/anoppa/Bit_Torrent_Project")
	aboutContainer := container.New(layout.NewVBoxLayout(),
		aboutLabel,
		widget.NewHyperlink("GitHub repository", repoURL),
	)
	content := container.New(layout.NewVBoxLayout(),
		aboutContainer,
		layout.NewSpacer(),
		buttons,
	)

	aboutWindow.SetContent(content)
	aboutWindow.Show()
}

func main() {

	application := app.NewWithID(appID)

	win := application.NewWindow(appName)
	win.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Menu",
			fyne.NewMenuItem("Settings", func() {
				settingsWindow := application.NewWindow("Fyne Settings")
				settingsWindow.SetContent(settings.NewSettings().LoadAppearanceScreen(settingsWindow))
				settingsWindow.Resize(fyne.NewSize(480, 480))
				settingsWindow.Show()
			}),
			fyne.NewMenuItem("About", func() {
				showAboutWindow(application)
			}),
		)), // a quit item will be appended to our first menu
	)
	win.SetMaster()
	win.CenterOnScreen()

	///tab INITIAL
	ipLabel := widget.NewLabel("Client IP:")
	ipEntry := widget.NewEntry()
	setIPButton := widget.NewButtonWithIcon("Set IP", theme.ConfirmIcon(), func() {
		ip := ipEntry.Text
		//source := torrentEntry.Text
		if net.ParseIP(ip) == nil {
			log.Println("Not reconognize that ip")
			dialog.ShowInformation("Set IP", "Failed", win)
			return
		}
		dialog.ShowInformation("Set IP", "Accept", win)
		ipEntry.Disable()
		// Pass to client
	})

	portLabel := widget.NewLabel("Client Port:")
	portEntry := widget.NewEntry()
	setPortButton := widget.NewButtonWithIcon("Set Port", theme.ConfirmIcon(), func() {
		port := portEntry.Text
		p2, err := strconv.Atoi(port)
		if err!=nil{
			log.Println("Not reconognize that port")
			dialog.ShowInformation("Set Port", "Failed", win)
			return
		}
		if p2 < 1024 || p2 > 65535{
			log.Println("Port out of range")
			dialog.ShowInformation("Set Port", "Failed valid range(1024-65535)", win)
			return
		}
		dialog.ShowInformation("Set Port", "Accept", win)
		portEntry.Disable()
		// Pass to client
	})

	ipButtons := container.NewHBox(setIPButton) //, chestFolderButton, currentFolderButton)
	portButtons := container.NewHBox(setPortButton)
	//downloadsButtons := container.NewCenter(downloadButton)

	config := container.NewVBox(
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				ipLabel,
				ipButtons,
			),
			ipLabel,
			ipEntry,
			ipButtons),
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				portLabel,
				portButtons,
			),
			portLabel,
			portEntry,
			portButtons),
		// container.New(
		// 	layout.NewBorderLayout(
		// 		nil,
		// 		nil,
		// 		nil,
		// 		downloadsButtons,
		// 	),
		// 	downloadsButtons),
	)

	//portLabel := widget.NewLabel("Client Port:")
	//portEntry := widget.NewEntry()

	//tab ONE DOWNLOADED

	//var (
	//	dest   string
	//	source string
	//)
	//dest = ""
	//source = ""

	folderLabel := widget.NewLabel("Output folder:")
	folderEntry := widget.NewEntry()
	//folderEntry.Disable()
	torrentLabel := widget.NewLabel("Select .torrent:")
	torrentEntry := widget.NewEntry()
	torrentOpenButton := widget.NewButtonWithIcon("Folder…", theme.DocumentSaveIcon(), func() {
		openDialog := dialog.NewFileOpen(
			func(r fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowInformation("Open Error", r.URI().Path(), win)
					return
				}
				if r == nil {
					// Cancelled
					return
				}
				torrentPath := strings.TrimPrefix(r.URI().Path(), "file://")
				if runtime.GOOS == "windows" {
					torrentPath = strings.ReplaceAll(torrentPath, "/", "\\")
				}
				torrentEntry.SetText(torrentPath)
				log.Println(torrentPath)
				//source = torrentPath
				//stat, err := os.Stat(r.URI().Path())
				dialog.ShowInformation("Selction ", "Success", win)
			},
			win,
		)
		openDialog.SetFilter(
			storage.NewExtensionFileFilter([]string{".pdf"}))
		openDialog.Show()
	})

	folderOpenButton := widget.NewButtonWithIcon("Folder…", theme.DocumentSaveIcon(), func() {
		open := dialog.NewFolderOpen(
			func(folder fyne.ListableURI, err error) {
				if err != nil {
					showErrorf(win, "Error when trying to select folder: %v", err)
					return
				}
				if folder == nil {
					// Cancelled
					return
				}
				folderPath := strings.TrimPrefix(folder.String(), "file://")
				if runtime.GOOS == "windows" {
					folderPath = strings.ReplaceAll(folderPath, "/", "\\")
				}
				log.Println(folderPath)
				folderEntry.SetText(folderPath)
				//dest = folderPath
				dialog.ShowInformation("Selction ", "Success", win)
			},
			win,
		)
		var uri fyne.ListableURI
		uri, err := storage.ListerForURI(storage.NewFileURI(folderEntry.Text))
		if err != nil {
			log.Printf("Couldn't get a listable URI for directory %s: %v", folderEntry.Text, err)
		} else {
			open.SetLocation(uri)
		}
		open.Show()
	})

	bar := widget.NewProgressBar()
	bar.Resize(fyne.NewSize(400, bar.MinSize().Height))
	bar.Hide()
	
	downloadButton := widget.NewButton("Start", func() {
		dest := folderEntry.Text
		source := torrentEntry.Text
		if dest == "" || source == "" {
			log.Println("Is necesary a destinantion and a source")
			dialog.ShowInformation("Start ", "Failed", win)
			return
		}
		log.Println(source + " ---> " + dest)
		//newProgressBar("progress", win).Show()
		bar.Show()
		go func() {
			for i := 0.0; i <= 1.0; i += 0.1 {
				time.Sleep(time.Millisecond * 500)
				bar.SetValue(i)
			}
		}()
		// Pass to client
	})

	folderButtons := container.NewHBox(folderOpenButton) //, chestFolderButton, currentFolderButton)
	torrentButtons := container.NewHBox(torrentOpenButton)
	downloadsButtons := container.NewCenter(downloadButton)

	download := container.NewVBox(
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				folderLabel,
				folderButtons,
			),
			folderLabel,
			folderEntry,
			folderButtons),
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				torrentLabel,
				torrentButtons,
			),
			torrentLabel,
			torrentEntry,
			torrentButtons),
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				nil,
				downloadsButtons,
			),
			downloadsButtons),
		container.NewGridWithRows(4,bar),
	)

	//Tab Two PUBLISH

	tab2torrentLabel := widget.NewLabel("Select .torrent:")
	tab2torrentEntry := widget.NewEntry()
	tab2torrentOpenButton := widget.NewButtonWithIcon("Folder…", theme.DocumentSaveIcon(), func() {
		openDialog := dialog.NewFileOpen(
			func(r fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowInformation("Open Error", r.URI().Path(), win)
					return
				}
				if r == nil {
					// Cancelled
					return
				}
				torrentPath := strings.TrimPrefix(r.URI().Path(), "file://")
				if runtime.GOOS == "windows" {
					torrentPath = strings.ReplaceAll(torrentPath, "/", "\\")
				}
				tab2torrentEntry.SetText(torrentPath)
				log.Println(torrentPath)
				//stat, err := os.Stat(r.URI().Path())
				dialog.ShowInformation("Selction ", "Success", win)
			},
			win,
		)
		openDialog.SetFilter(
			storage.NewExtensionFileFilter([]string{".torrent"}))
		openDialog.Show()
	})

	tab2torrentButtons := container.NewHBox(tab2torrentOpenButton)

	publish := container.NewVBox(
		container.New(
			layout.NewBorderLayout(
				nil,
				nil,
				tab2torrentLabel,
				tab2torrentButtons,
			),
			tab2torrentLabel,
			tab2torrentEntry,
			tab2torrentButtons),
	)

	tabItems := make([]*container.TabItem, 0, 2)

	tabItems = append(tabItems, container.NewTabItem("Config", config))
	tabItems = append(tabItems, container.NewTabItem("Download", download))
	tabItems = append(tabItems, container.NewTabItem("Publish", publish))
	tabs := container.NewAppTabs(tabItems...)
	tabs.SetTabLocation(container.TabLocationLeading)

	win.SetContent(
		container.New(
			layout.NewPaddedLayout(),
			//generalOptions,
			tabs,
		),
	)

	if _, ok := application.Driver().(desktop.Driver); ok {
		// Desktop only
		quitShortcut := desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: desktop.ControlModifier}
		win.Canvas().AddShortcut(&quitShortcut, func(shortcut fyne.Shortcut) {
			application.Quit()
		})
	}

	win.Resize(fyne.NewSize(800, 600))

	win.ShowAndRun()
}
