package main

import (
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var bgColor = tcell.ColorDefault

var history = []string{}
var historyIdx = 0

func setBoxAttr(box *tview.Box, title string) {
	box.SetBorder(true)
	box.SetTitleAlign(tview.AlignLeft)
	box.SetTitle(title)
	box.SetBackgroundColor(bgColor)
	box.SetTitleColor(bgColor)
	box.SetBorderColor(bgColor)
}

func drawSidebar() (*tview.Grid, *tview.TextView, *tview.TextView, *tview.TextView) {
	sidebarGrid := tview.NewGrid().SetRows(-1, -1, -1).SetBorders(false)
	roomInfoView := tview.NewTextView().SetDynamicColors(true)
	roomInfoView.SetBackgroundColor(bgColor)
	roomInfoView.SetTextColor(bgColor)
	setBoxAttr(roomInfoView.Box, "RoomInfo")

	chatView := tview.NewTextView().SetDynamicColors(true)
	chatView.SetBackgroundColor(bgColor)
	chatView.SetTextColor(bgColor)
	setBoxAttr(chatView.Box, "Chat")

	infoView := tview.NewTextView().SetDynamicColors(true)
	infoView.SetBackgroundColor(bgColor)
	infoView.SetTextColor(bgColor)
	setBoxAttr(infoView.Box, "Info")

	sidebarGrid.
		AddItem(roomInfoView, 0, 0, 1, 1, 0, 0, false).
		AddItem(chatView, 1, 0, 1, 1, 0, 0, false).
		AddItem(infoView, 2, 0, 1, 1, 0, 0, false)

	return sidebarGrid, roomInfoView, chatView, infoView
}

func drawMainPanel() (*tview.Grid, *tview.TextView, *tview.TextView, *tview.InputField) {
	mainPanelGrid := tview.NewGrid().SetRows(-1, 6, 3).SetBorders(false)
	messagesView := tview.NewTextView().SetDynamicColors(true)
	messagesView.SetBackgroundColor(bgColor)
	messagesView.SetTextColor(bgColor)
	setBoxAttr(messagesView.Box, "Messages")

	statusView := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetToggleHighlights(true)
	statusView.SetBackgroundColor(bgColor)
	statusView.SetTextColor(bgColor)
	setBoxAttr(statusView.Box, "Status")

	// setBoxAttr(infoView.Box, "Info")

	input := tview.NewInputField()
	input.SetFormAttributes(0, tcell.ColorDefault, bgColor, tcell.ColorDefault, bgColor)
	setBoxAttr(input.Box, "Send")

	mainPanelGrid.
		AddItem(messagesView, 0, 0, 1, 1, 0, 0, false).
		AddItem(statusView, 1, 0, 1, 1, 0, 0, false).
		AddItem(input, 2, 0, 1, 1, 0, 0, true)
	return mainPanelGrid, messagesView, statusView, input
}

func draw(app *tview.Application) *tview.Grid {
	sidebarGrid, roomInfoView, chatView, infoView := drawSidebar()
	mainPanelGrid, messagesView, statusView, input := drawMainPanel()
	rootGrid := tview.NewGrid().SetColumns(-3, -5).SetBorders(false)
	rootGrid.
		AddItem(sidebarGrid, 0, 0, 1, 1, 0, 0, false).
		AddItem(mainPanelGrid, 0, 1, 1, 1, 0, 0, true)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			msg := input.GetText()
			sendChan <- msg

			historyIdx = len(history)
			input.SetText("")
		}
	})

	input.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 || currentText[0] != '/' {
			return
		}
		cmds := []string{"/ready (ready for game)", "/use card1 card2.. (play selected cards) ", "/pass (pass current turn)", "/quit (quit the game)"}
		for _, entry := range cmds {
			if strings.HasPrefix(entry, currentText) {
				entries = append(entries, entry)
			}
		}
		return
	})
	input.SetPlaceholder(" Type / to start a command")
	input.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorDarkGray))
	input.SetAutocompleteStyles(
		tcell.ColorLightGray,
		tcell.StyleDefault.Foreground(tcell.ColorBlack),
		tcell.StyleDefault.Background(tcell.ColorDimGray).Foreground(tcell.ColorWhite),
	)
	input.SetAutocompletedFunc(func(text string, index, source int) bool {
		if source != tview.AutocompletedNavigate {
			cmd := strings.Split(text, " ")[0]
			input.SetText(cmd)
		}
		return source == tview.AutocompletedEnter || source == tview.AutocompletedClick
	})

	go handleMessages(app, messagesView, roomInfoView, statusView, chatView, infoView)

	return rootGrid
}

func Run(app *tview.Application) {
	if err := app.SetRoot(draw(app), true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}

func handleMessages(
	app *tview.Application,
	messagesView *tview.TextView,
	roomInfoView *tview.TextView,
	statusView *tview.TextView,
	chatView *tview.TextView,
	infoView *tview.TextView,
) {
	for message := range msgChan {
		switch message.MsgType {
		case MSG_MESSAGE:
			history = append(history, message.Content)
			log.Println(message.Content)
			messagesView.SetText(strings.Join(history, "\n"))
			messagesView.ScrollToEnd()
		case MSG_ERROR:
			log.Println(message.Content)
		case MSG_PLAYER_STATUS:
			statusMsgs := strings.Split(message.Content, "_")
			position := statusMsgs[0]
			cards := statusMsgs[1]
			log.Println(cards)
			statusStr := "Position: " + position + "\nCards: " + cards
			highlights := statusView.GetHighlights()
			log.Println(highlights)
			statusView.SetText(statusStr).SetChangedFunc(func() {
				app.Draw()
			})
			statusView.Highlight(highlights...)

		case MSG_INFO:
			infoView.SetText(infoView.GetText(false) + " " + message.Content + "\n")
			infoView.ScrollToEnd()
		case MSG_CHAT:
			// log.Println(message.Content)
			chatView.SetText(chatView.GetText(false) + message.Content + "\n")
			chatView.ScrollToEnd()
		case MSG_ROOM_INFO:
			log.Println(message.Content)
			roomInfoMsgs := strings.Split(message.Content, "_")
			roomInfoStr := "Status: " + roomInfoMsgs[0] + "\nPlayers:\n" + roomInfoMsgs[1]
			roomInfoView.SetText(roomInfoStr)
		case MSG_STOP:
			history = append(history, message.Content)
			log.Println(message.Content)
			messagesView.SetText(strings.Join(history, "\n"))
			messagesView.ScrollToEnd()
			app.Draw()
			time.Sleep(1 * time.Second)
			app.Stop()
		}
		app.Sync().Draw()
	}
}
