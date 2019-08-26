package cli

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func drawTitle() {
	termWidth, _ := ui.TerminalDimensions()

	title := widgets.NewParagraph()
	title.Text = "Unfork.io"
	title.TextStyle.Fg = ui.ColorWhite
	title.TextStyle.Bg = ui.ColorClear
	title.TextStyle.Modifier = ui.ModifierBold
	title.Border = false
	title.SetRect(termWidth/2-10, 0, termWidth/2+10, 1)
	ui.Render(title)

	replicated := widgets.NewParagraph()
	replicated.Text = "Replicated      Helm"
	replicated.TextStyle.Fg = ui.ColorWhite
	replicated.TextStyle.Bg = ui.ColorClear
	replicated.Border = false
	replicated.SetRect(termWidth/2-15, 1, termWidth/2+15, 2)
	ui.Render(replicated)

	heart := widgets.NewParagraph()
	heart.Text = "❤"
	heart.TextStyle.Fg = ui.ColorRed
	heart.TextStyle.Bg = ui.ColorClear
	heart.Border = false
	heart.SetRect(termWidth/2-3, 1, termWidth/2+1, 2)
	ui.Render(heart)
}
