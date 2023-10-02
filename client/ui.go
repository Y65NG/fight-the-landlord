package main

import (
	"log"
	"strings"

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

	helperView := tview.NewTextView().SetDynamicColors(true)
	helperView.SetBackgroundColor(bgColor)
	helperView.SetTextColor(bgColor)
	setBoxAttr(helperView.Box, "Help")

	sidebarGrid.
		AddItem(roomInfoView, 0, 0, 1, 1, 0, 0, false).
		AddItem(chatView, 1, 0, 1, 1, 0, 0, false).
		AddItem(helperView, 2, 0, 1, 1, 0, 0, false)
	return sidebarGrid, roomInfoView, chatView, helperView
}

func drawMainPanel() (*tview.Grid, *tview.TextView, *tview.TextView, *tview.TextView, *tview.InputField) {
	mainPanelGrid := tview.NewGrid().SetRows(-1, 6, 3, 3).SetBorders(false)
	messagesView := tview.NewTextView().SetDynamicColors(true)
	messagesView.SetBackgroundColor(bgColor)
	messagesView.SetTextColor(bgColor)
	setBoxAttr(messagesView.Box, "Messages")

	statusView := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetToggleHighlights(true)
	statusView.SetBackgroundColor(bgColor)
	statusView.SetTextColor(bgColor)
	setBoxAttr(statusView.Box, "Status")

	infoView := tview.NewTextView().SetDynamicColors(true)
	infoView.SetBackgroundColor(bgColor)
	infoView.SetTextColor(bgColor)
	// setBoxAttr(infoView.Box, "Info")

	input := tview.NewInputField()
	input.SetFormAttributes(0, tcell.ColorDefault, bgColor, tcell.ColorDefault, bgColor)
	setBoxAttr(input.Box, "Send")

	mainPanelGrid.
		AddItem(messagesView, 0, 0, 1, 1, 0, 0, false).
		AddItem(statusView, 1, 0, 1, 1, 0, 0, false).
		AddItem(infoView, 2, 0, 1, 1, 0, 0, false).
		AddItem(input, 3, 0, 1, 1, 0, 0, true)
	return mainPanelGrid, messagesView, statusView, infoView, input
}

func draw(app *tview.Application) *tview.Grid {
	sidebarGrid, roomInfoView, chatView, helperView := drawSidebar()
	helperView.SetText(
		`Type slash (/) to start a command.
Available commands:
- /ready: be ready for the game
- /use card1 card2...: use the cards you selected
- /pass: pass your current turn
- /quit: quit the game`)
	mainPanelGrid, messagesView, statusView, infoView, input := drawMainPanel()
	rootGrid := tview.NewGrid().SetColumns(-1, -2).SetBorders(false)
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

	go handleMessages(app, messagesView, roomInfoView, statusView, chatView, infoView)

	return rootGrid
}

func Run() {
	app := tview.NewApplication()

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
		}
		app.Draw()
	}
}
