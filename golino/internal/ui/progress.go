package ui

import (
    "sync"

    "github.com/pterm/pterm"
)

type Progress struct {
    pb   *pterm.ProgressbarPrinter
    once sync.Once
}

func NewProgress() *Progress {
    pterm.EnableDebugMessages()
    return &Progress{}
}

func (p *Progress) Start(total int, title string) {
    p.once.Do(func() {})
    p.pb, _ = pterm.DefaultProgressbar.WithTotal(total).WithTitle(title).Start()
}

func (p *Progress) Increment(msg string) {
    if p.pb != nil {
        p.pb.UpdateTitle(msg)
        p.pb.Increment()
    }
}

func (p *Progress) Stop() {
    if p.pb != nil {
        p.pb.Stop()
    }
}

func (p *Progress) Close() {
    p.Stop()
}
