package main

import (
	"Bit_Torrent_Project/Client/client/torrent_file"
	"Bit_Torrent_Project/Client/torrent_peer"
	"Bit_Torrent_Project/Client/utils"
	"fmt"
	"log"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thanhpk/randstr"

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

	var wg = sync.WaitGroup{}

	const bootstrapPort = "6888"
	peerId := ""
	info := utils.LoadInfo("./info.json")
	if info.PeerId != "" {
		peerId = info.PeerId
	} else {
		peerId = randstr.Hex(10)
	}
	errChan := make(chan error)

	var torrentPath, downloadTo string
	torrentPath = ""
	downloadTo = ""
	IP := ""
	port := ""

	// Starting server
	csServer := &torrent_peer.ConnectionsState{LastUpload: map[string]time.Time{}, NumberOfBlocksInLast30Seconds: map[string]int{}}
	wg.Add(1)
	go func() {
		utils.StartClientUploader(info, peerId, IP, csServer, errChan)
		wg.Done()
	}()

	//////UI

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
		IP = ip
	})

	portLabel := widget.NewLabel("Client Port:")
	portEntry := widget.NewEntry()
	setPortButton := widget.NewButtonWithIcon("Set Port", theme.ConfirmIcon(), func() {
		port2 := portEntry.Text
		p2, err := strconv.Atoi(port2)
		if err != nil {
			log.Println("Not reconognize that port")
			dialog.ShowInformation("Set Port", "Failed", win)
			return
		}
		if p2 < 1024 || p2 > 65535 {
			log.Println("Port out of range")
			dialog.ShowInformation("Set Port", "Failed valid range(1024-65535)", win)
			return
		}
		dialog.ShowInformation("Set Port", "Accept", win)
		portEntry.Disable()
		// Pass to client
		port = port2
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
	)

	//tab ONE DOWNLOADED

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
				dialog.ShowInformation("Selection ", "Success", win)
				
			},
			win,
		)
		openDialog.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
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
		if net.ParseIP(IP) == nil {
			log.Println("Necesary a correct IP")
			dialog.ShowInformation("IP", "Incorret", win)
			ipEntry.Enable()
			return
		}
		p2, err := strconv.Atoi(port)
		if err != nil {
			log.Println("Not reconognize that port")
			dialog.ShowInformation("Port", "Incorrect", win)
			portEntry.Enable()
			return
		}
		if p2 < 1024 || p2 > 65535 {
			log.Println("Port out of range")
			dialog.ShowInformation("Port", "Failed valid range(1024-65535)", win)
			portEntry.Enable()
			return
		}

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
		wg.Add(1)
		go func() {
			pieces, infoHash, bitfield, err := utils.Download(torrentPath, downloadTo, IP, peerId, nil)
			if err != nil {
				log.Println(err)
				return
			}
			utils.SaveToInfo(torrentPath, pieces, infoHash, []byte(bitfield))
			wg.Done()
		}()

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
		container.NewGridWithRows(4, bar),
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
				dialog.ShowInformation("Selection ", "Success", win)
				filePath := torrentPath
				tPath := filePath[:len(filePath)-5] + ".torrent"
				trackerUrl := "192.168.169.32:8167"
				torrent_file.BuildTorrentFile(filePath, tPath, trackerUrl)
				utils.Publish(filePath, peerId, IP)
			},
			win,
		)
		// openDialog.SetFilter(
		// 	storage.NewExtensionFileFilter([]string{".torrent"}))
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

	//Publish("../b.torrent", peerId, "192.168.169.14")
	// _ = torrent_file.BuildTorrentFile("../VID-20211202-WA0190.mp4", "../b.torrent", "192.168.169.32:8167")

	//bootstrapADDR := IP + ":" + bootstrapPort
	//
	//// Starting DHT
	//options := &dht.Options{
	//	ID:                   []byte(peerId),
	//	IP:                   IP,
	//	Port:                 6881,
	//	ExpirationTime:       time.Hour,
	//	RepublishTime:        time.Minute * 50,
	//	TimeToDie:            time.Second * 6,
	//	TimeToRefreshBuckets: time.Minute * 15,
	//}
	//dhtNode := dht.NewDHT(options)
	//exitChan := make(chan string)
	//
	//go dhtNode.RunServer(exitChan)
	//
	//time.Sleep(time.Second * 2)

	//dhtNode.JoinNetwork(bootstrapADDR)

	// Starting downloader

	var downloadWord string
	_, err := fmt.Scanln(&downloadWord)
	if err != nil {
		log.Fatal(err)
	}
	if downloadWord == "d" {
		fmt.Println("Waiting for paths to download...")
		wg.Add(1)
		go func() {
			pieces, infoHash, bitfield, err := utils.Download(torrentPath, downloadTo, IP, peerId, nil)
			if err != nil {
				log.Println(err)
				return
			}
			utils.SaveToInfo(torrentPath, pieces, infoHash, []byte(bitfield))
			wg.Done()
		}()
	}

	//go func() {
	//	for {
	//		if _, ok := <-exitChan; ok {
	//			go dhtNode.RunServer(exitChan)
	//		}
	//	}
	//}()

	wg.Wait()

	//defer close(exitChan)

}
