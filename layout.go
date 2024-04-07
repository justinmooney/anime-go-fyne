package main

import (
	"fyne.io/fyne/v2"
)

const (
	leftPanelWidth  = 400
	searchBoxHeight = 30
)

type myLayout struct {
	searchBox, list, content fyne.CanvasObject
}

func newLayout(searchBox, list, content fyne.CanvasObject) fyne.Layout {
	return &myLayout{searchBox: searchBox, list: list, content: content}
}

func (l *myLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	searchBoxHeight := l.searchBox.MinSize().Height
	l.searchBox.Resize(fyne.NewSize(leftPanelWidth, searchBoxHeight))

	l.list.Move(fyne.NewPos(0, searchBoxHeight))
	l.list.Resize(fyne.NewSize(leftPanelWidth, size.Height-searchBoxHeight))

	l.content.Move(fyne.NewPos(leftPanelWidth, 0))
	l.content.Resize(fyne.NewSize(size.Width-leftPanelWidth, size.Height))
}

func (l *myLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1400, 800)
}
