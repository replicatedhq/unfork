package cli

import (
	"fmt"
	"os"
	"path"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"

	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/replicatedhq/unfork/pkg/unforker"
	"github.com/replicatedhq/unfork/pkg/util"
)

var (
	responsiveBreakpoint = 300
)

type Home struct {
	chartsTable *widgets.Table
	isListening bool

	chartHeaderNarrow []string
	chartHeaderWide   []string

	uiCh chan unforker.UIEvent

	localCharts     []*unforker.LocalChart
	upstreamMatches []chartindex.ChartMatch

	selectedChartIndex       int
	selectedUpstreamIndex    int
	showUnfork               bool
	isUnforking              bool
	needsOverwritePermission bool
	dialogMessage            string

	focusPane string
}

func createHome(uiCh chan unforker.UIEvent) *Home {
	home := Home{}

	home.chartHeaderNarrow = []string{"Helm Chart", "Chart Version"}
	home.chartHeaderWide = []string{"Helm Chart", "Namespace", "Installed App Version", "Installed Chart Version"}

	home.uiCh = uiCh
	home.localCharts = []*unforker.LocalChart{}

	home.focusPane = "charts"

	return &home
}

func (h *Home) render() error {
	termWidth, termHeight := ui.TerminalDimensions()

	drawTitle()

	border := widgets.NewParagraph()
	border.SetRect(termWidth/2-2, 3, termWidth, termHeight-1)
	ui.Render(border)

	h.chartsTable = widgets.NewTable()
	h.chartsTable.TextStyle = ui.NewStyle(ui.ColorCyan)
	h.chartsTable.FillRow = true
	if termWidth > responsiveBreakpoint {
		h.chartsTable.Rows = h.wideCharts()
	} else {
		h.chartsTable.Rows = h.narrowCharts()
	}
	h.chartsTable.SetRect(0, 2, termWidth/2, termHeight)
	h.chartsTable.RowStyles[0] = ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)
	ui.Render(h.chartsTable)

	h.drawSelectedChart()

	if h.showUnfork {
		h.drawUnfork()
	}

	// find the minimum width of rectangle that will fit the text on one line
	// TODO: handle multiline
	if h.dialogMessage != "" {
		overwritePrompt := widgets.NewParagraph()
		promptY := termHeight/2 - 2
		promptLen := len(h.dialogMessage)
		promptXMin := termWidth/2 - (promptLen / 2)
		promptXMax := termWidth/2 + (promptLen / 2)
		overwritePrompt.SetRect(promptXMin, promptY, promptXMax, promptY+3)
		for overwritePrompt.Inner.Dx() < promptLen {
			promptXMin--
			promptXMax++
			overwritePrompt.SetRect(promptXMin, promptY, promptXMax, promptY+3)
		}

		overwritePrompt.Text = h.dialogMessage
		ui.Render(overwritePrompt)
	}

	if !h.isListening {
		h.isListening = true

		go func() {
			for {
				select {
				case uiEvent := <-h.uiCh:
					if uiEvent.EventName == "new_chart" {
						chart := uiEvent.Payload.(*unforker.LocalChart)
						h.localCharts = append(h.localCharts, chart)

						if termWidth > responsiveBreakpoint {
							h.chartsTable.Rows = h.wideCharts()
						} else {
							h.chartsTable.Rows = h.narrowCharts()
						}

						ui.Render(h.chartsTable)
					}
				}
			}
		}()
	}

	return nil
}

func (h *Home) handleEvent(e ui.Event) (bool, error) {
	switch e.ID {
	case "<Escape>", "q", "<C-c>":
		if h.showUnfork {
			h.showUnfork = false
			h.needsOverwritePermission = false
			h.dialogMessage = ""
			ui.Clear()
			err := h.render()
			if err != nil {
				return false, errors.Wrapf(err, "render event %q", e.ID)
			}
		} else {
			return true, nil
		}
	case "<Resize>":
		ui.Clear()
		err := h.render()
		if err != nil {
			return false, errors.Wrapf(err, "render event %q", e.ID)
		}
	case "<Down>", "s":
		if !h.showUnfork && !h.isUnforking {
			if h.focusPane == "charts" {
				if h.selectedChartIndex == -1 {
					h.selectedChartIndex = 1
				} else if h.selectedChartIndex < len(h.chartsTable.Rows)-1 {
					h.selectedChartIndex++
				} else {
					h.selectedChartIndex = 1
				}
				if err := h.highlightChart(); err != nil {
					return false, err
				}
			} else if h.focusPane == "upstreams" {
				h.highlightNextUpstream()
			}
		}
	case "<Up>", "w":
		if !h.showUnfork && !h.isUnforking {
			if h.focusPane == "charts" {
				if h.selectedChartIndex == -1 {
					h.selectedChartIndex = 1
				} else if h.selectedChartIndex > 1 {
					h.selectedChartIndex--
				} else {
					h.selectedChartIndex = len(h.chartsTable.Rows) - 1
				}
				if err := h.highlightChart(); err != nil {
					return false, err
				}
			} else if h.focusPane == "upstreams" {
				h.highlightPreviousUpstream()
			}
		}
	case "<Right>", "d":
		if !h.showUnfork && !h.isUnforking {
			if h.focusPane == "charts" {
				h.focusPane = "upstreams"
				ui.Clear()
				h.render()
			}
		}
	case "<Left>", "a":
		if !h.showUnfork && !h.isUnforking {
			if h.focusPane == "upstreams" {
				h.focusPane = "charts"
				ui.Clear()
				h.render()
				h.highlightChart()
			}
		}
	case "<Enter>":
		if h.showUnfork && !h.isUnforking {
			h.isUnforking = true

			unforkPath := h.findUnforkPath()
			_, err := os.Stat(unforkPath)
			if !os.IsNotExist(err) {
				// dir exists, prompt for user to choose whether to use a different name or overwrite
				h.needsOverwritePermission = true
				h.dialogMessage = fmt.Sprintf(" Should %q be overwritten? (y/n) ", unforkPath)
				ui.Clear()
				h.render()
			}
			if !h.needsOverwritePermission {
				if err := h.doUnfork(); err != nil {
					panic(err)
				}
			}
		} else if !h.isUnforking {
			if h.focusPane == "upstreams" {
				if h.selectedUpstreamIndex == 0 {
					break
				}

				h.showUnfork = true
				ui.Clear()
				h.render()
			}
		}
	case "y", "Y":
		if h.needsOverwritePermission {
			h.needsOverwritePermission = false
			// overwrite dir and run unfork
			unforkPath := h.findUnforkPath()
			if err := os.RemoveAll(unforkPath); err != nil {
				panic(err)
			}

			if err := h.doUnfork(); err != nil {
				panic(err)
			}
		}
	case "n", "N":
		if h.needsOverwritePermission {
			h.needsOverwritePermission = false
			// don't overwrite dir, just run unfork

			if err := h.doUnfork(); err != nil {
				panic(err)
			}
		}
	}

	return false, nil
}

func (h *Home) doUnfork() error {

	h.dialogMessage = "unforking..."
	ui.Clear()
	h.render()

	localChart := h.localCharts[h.selectedChartIndex-1]
	upstreamChart := h.upstreamMatches[h.selectedUpstreamIndex-1]

	unforkedDir, err := unforker.Unfork(localChart, upstreamChart)
	if err != nil {
		return err
	}

	h.isUnforking = false
	h.dialogMessage = fmt.Sprintf(" Unforked to %q. Press 'q' to return to the main screen ", unforkedDir)
	ui.Clear()
	h.render()

	return nil
}

func (h *Home) findUnforkPath() string {
	localChart := h.localCharts[h.selectedChartIndex-1]
	return path.Join(util.HomeDir(), localChart.HelmName)
}

func (h *Home) highlightChart() error {
	ui.Clear()
	err := h.render()
	if err != nil {
		return errors.Wrapf(err, "render chart to highlight")
	}

	for i := range h.chartsTable.Rows {
		if i == 0 {
			continue
		}

		if i != h.selectedChartIndex {
			h.chartsTable.RowStyles[i] = ui.NewStyle(ui.ColorBlue, ui.ColorClear)
		} else {
			h.chartsTable.RowStyles[i] = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
		}
	}
	ui.Render(h.chartsTable)

	return nil
}

func (h *Home) wideCharts() [][]string {
	rows := [][]string{h.chartHeaderWide}

	for _, localChart := range h.localCharts {
		rows = append(rows, []string{
			localChart.ChartName,
			"default",
			localChart.AppVersion,
			localChart.ChartVersion,
		})
	}

	return rows
}

func (h *Home) narrowCharts() [][]string {
	rows := [][]string{h.chartHeaderNarrow}

	for _, localChart := range h.localCharts {
		rows = append(rows, []string{
			localChart.ChartName,
			localChart.ChartVersion,
		})
	}

	return rows
}

func (h *Home) highlightNextUpstream() {
	if h.selectedUpstreamIndex < len(h.upstreamMatches) {
		h.selectedUpstreamIndex++
	} else {
		h.selectedUpstreamIndex = 1
	}
	h.drawSelectedChart()
}

func (h *Home) highlightPreviousUpstream() {
	if h.selectedUpstreamIndex > 1 {
		h.selectedUpstreamIndex--
	} else {
		h.selectedUpstreamIndex = len(h.upstreamMatches) - 1
	}
	h.drawSelectedChart()
}

func (h *Home) drawSelectedChart() {
	if h.selectedChartIndex == 0 || h.selectedChartIndex > len(h.localCharts) {
		return
	}

	termWidth, termHeight := ui.TerminalDimensions()
	ourLeft := termWidth / 2
	ourRight := termWidth - 1
	ourTop := 4
	ourBottom := termHeight - 4

	chartName := widgets.NewParagraph()
	chartName.Border = false
	chartName.Title = fmt.Sprintf("Chart Name: %s", h.localCharts[h.selectedChartIndex-1].ChartName)
	chartName.SetRect(ourLeft, ourTop, ourRight, ourTop+1)
	ui.Render(chartName)

	chartVersion := widgets.NewParagraph()
	chartVersion.Border = false
	chartVersion.Title = fmt.Sprintf(" Your Chart Version: %s", h.localCharts[h.selectedChartIndex-1].ChartVersion)
	chartVersion.SetRect(ourLeft, ourTop+1, ourRight, ourTop+2)
	ui.Render(chartVersion)

	appVersion := widgets.NewParagraph()
	appVersion.Border = false
	appVersion.Title = fmt.Sprintf(" Your App Version: %s", h.localCharts[h.selectedChartIndex-1].AppVersion)
	appVersion.SetRect(ourLeft, ourTop+2, ourRight, ourTop+3)
	ui.Render(appVersion)

	possibleUpstreams := widgets.NewParagraph()
	possibleUpstreams.Border = false
	if h.focusPane == "charts" {
		possibleUpstreams.Title = "Possible Upstream Helm Charts (press → to select)"
	} else if h.focusPane == "upstreams" {
		possibleUpstreams.Title = "Possible Upstream Helm Charts (↑ ↓ to select)"
	}
	possibleUpstreams.SetRect(ourLeft, ourTop+4, ourRight, ourTop+5)
	ui.Render(possibleUpstreams)

	upstreamsTable := widgets.NewTable()
	upstreamsTable.FillRow = true
	upstreamsTable.SetRect(ourLeft, ourTop+5, ourRight, ourBottom-2)
	upstreamsTable.RowStyles[0] = ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)
	upstreamsTable.Rows = [][]string{
		[]string{"Repo/Chart", "Closest Version", "Latest Chart/App Version"},
	}

	localChart := h.localCharts[h.selectedChartIndex-1]
	upstreamMatches, err := chartindex.FindBestUpstreamMatches(localChart.ChartName, localChart.ChartVersion, localChart.AppVersion)
	if err != nil {
		return
	}
	h.upstreamMatches = upstreamMatches

	if h.selectedUpstreamIndex == 0 {
		if len(upstreamMatches) > 0 {
			h.selectedUpstreamIndex = 1
		}
	}

	for _, upstreamMatch := range upstreamMatches {
		upstreamsTable.Rows = append(upstreamsTable.Rows, []string{
			fmt.Sprintf("%s/%s", upstreamMatch.Repo, upstreamMatch.Name),
			upstreamMatch.ChartVersion,
			fmt.Sprintf("%s/%s", upstreamMatch.LatestChartVersion, upstreamMatch.LatestAppVersion),
		})
	}

	if h.focusPane == "upstreams" {
		for i := range upstreamsTable.Rows {
			if i == 0 {
				continue
			}
			if i != h.selectedUpstreamIndex {
				upstreamsTable.RowStyles[i] = ui.NewStyle(ui.ColorBlue, ui.ColorClear)
			} else {
				upstreamsTable.RowStyles[i] = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
			}
		}
	}

	ui.Render(upstreamsTable)
}

func (h *Home) drawUnfork() {
	termWidth, termHeight := ui.TerminalDimensions()
	ourLeft := 6
	ourRight := termWidth - 6
	ourTop := 4
	ourBottom := termHeight - 2

	if ourRight-ourLeft > 200 {
		ourLeft = termWidth/2 - 150
		ourRight = termWidth/2 + 150
	}

	ourCenter := (ourLeft + ourRight) / 2

	modal := widgets.NewParagraph()
	modal.Border = true
	modal.SetRect(ourLeft, ourTop, ourRight, ourBottom)
	ui.Render(modal)

	modalTitle := widgets.NewParagraph()
	modalTitle.Text = "Unfork"
	modalTitle.TextStyle.Fg = ui.ColorWhite
	modalTitle.TextStyle.Bg = ui.ColorClear
	modalTitle.TextStyle.Modifier = ui.ModifierBold
	modalTitle.Border = false
	modalTitle.SetRect(ourCenter-len(modalTitle.Text), ourTop+1, ourCenter+len(modalTitle.Text), ourTop+2)
	ui.Render(modalTitle)

	localChart := h.localCharts[h.selectedChartIndex-1]
	upstreamChart := h.upstreamMatches[h.selectedUpstreamIndex-1]

	whatsAboutToHappen := widgets.NewParagraph()
	whatsAboutToHappen.Text = fmt.Sprintf(
		`Press <Enter> to compare your local chart (%s@%s) with the upstream chart (%s/%s@%s) to create Kustomize compatible patches on disk`,
		localChart.ChartName, localChart.ChartVersion,
		upstreamChart.Repo, upstreamChart.Name, upstreamChart.ChartVersion)
	whatsAboutToHappen.TextStyle.Fg = ui.ColorWhite
	whatsAboutToHappen.TextStyle.Bg = ui.ColorClear
	whatsAboutToHappen.Border = false
	whatsAboutToHappen.SetRect(ourLeft+1, ourTop+4, ourRight-1, ourTop+8)
	ui.Render(whatsAboutToHappen)

	if h.isUnforking {
		unforking := widgets.NewParagraph()
		unforking.Text = "unforking"
		unforking.TextStyle.Fg = ui.ColorGreen
		unforking.TextStyle.Bg = ui.ColorClear
		unforking.TextStyle.Modifier = ui.ModifierBold
		unforking.Border = false
		unforking.SetRect(ourCenter-len(modalTitle.Text), ourTop+12, ourCenter+len(modalTitle.Text), ourTop+13)
		ui.Render(unforking)
	}

}
