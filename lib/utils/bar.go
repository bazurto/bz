// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package utils

import (
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type PB struct {
	progress *mpb.Progress
	bar      *mpb.Bar
}

func NewProgressBar(total int64) *PB {
	progress := mpb.New(mpb.WithWidth(64))
	bar := progress.AddBar(total,
		mpb.PrependDecorators(decor.Counters(decor.UnitKiB, "% .1f / % .1f")),
		mpb.AppendDecorators(decor.Percentage()),
	)
	return &PB{progress: progress, bar: bar}
}

func (o *PB) Set(v int64) {
	o.bar.SetCurrent(v)
}

func (o *PB) Wait() {
	o.progress.Wait()
}

func (o *PB) Done() {
	o.progress.Shutdown()
}
