package main

import (
	"fyne.io/fyne/v2"
)

const (
	leftPanelWidth    = 400
	searchBoxHeight   = 30
	textContentHeight = 300
)

type myLayout struct {
	searchBox, list, content, image fyne.CanvasObject
}

func newLayout(searchBox, list, content, image fyne.CanvasObject) fyne.Layout {
	return &myLayout{searchBox: searchBox, list: list, content: content, image: image}
}

func (l *myLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	searchBoxHeight := l.searchBox.MinSize().Height
	l.searchBox.Resize(fyne.NewSize(leftPanelWidth, searchBoxHeight))

	l.list.Move(fyne.NewPos(0, searchBoxHeight))
	l.list.Resize(fyne.NewSize(leftPanelWidth, size.Height-searchBoxHeight))

	l.content.Move(fyne.NewPos(leftPanelWidth, 0))
	l.content.Resize(fyne.NewSize(size.Width-leftPanelWidth, textContentHeight))

	l.image.Move(fyne.NewPos(leftPanelWidth, textContentHeight))
	l.image.Resize(fyne.NewSize(size.Width-leftPanelWidth, size.Height-textContentHeight))
}

func (l *myLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1400, 1000)
}
