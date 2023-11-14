// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/amzn/ion-go/ion"
)

type Version struct {
	nums     []int
	pre      string
	meta     string
	original string
}

func (o *Version) MarshalJSON() ([]byte, error) {
	// ion.Marshaler
	var strNums []string
	for _, n := range o.nums {
		strNums = append(strNums, strconv.Itoa(n))
	}

	// 0.0.0
	str := strings.Join(strNums, ".")

	if o.pre != "" {
		str = fmt.Sprintf("%s-%s", str, o.pre)
	}

	if o.meta != "" {
		str = fmt.Sprintf("%s+%s", str, o.meta)
	}

	return json.Marshal(str)
}

func (o *Version) MarshalIon(w ion.Writer) error {
	// ion.Marshaler
	var strNums []string
	for _, n := range o.nums {
		strNums = append(strNums, strconv.Itoa(n))
	}

	// 0.0.0
	str := strings.Join(strNums, ".")

	if o.pre != "" {
		str = fmt.Sprintf("%s-%s", str, o.pre)
	}

	if o.meta != "" {
		str = fmt.Sprintf("%s+%s", str, o.meta)
	}

	return ion.MarshalTo(w, str)
}

func (o *Version) UnmarshalJSON(data []byte) error {
	var tmp string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	v := NewVersion(tmp)
	o.nums = v.nums
	o.pre = v.pre
	o.meta = v.meta
	o.original = v.original
	return nil
}
func (o *Version) UnmarshalIon(r ion.Reader) error {
	var tmp string
	if err := ion.UnmarshalFrom(r, &tmp); err != nil {
		return err
	}
	v := NewVersion(tmp)
	o.nums = v.nums
	o.pre = v.pre
	o.meta = v.meta
	o.original = v.original
	return nil
}

func (o *Version) Canonical() string {
	var buf bytes.Buffer

	var nums []string
	for _, n := range o.nums {
		nums = append(nums, strconv.Itoa(n))
	}
	fmt.Fprintf(&buf, "%s", strings.Join(nums, "."))

	if o.pre != "" {
		fmt.Fprintf(&buf, "-%s", o.pre)
	}

	return buf.String()
}

func (o *Version) Original() string {
	return o.original
}

func (o *Version) String() string {
	if o.meta != "" {
		return fmt.Sprintf("%s+%s", o.Canonical(), o.meta)
	}
	return o.Canonical()

}

func NewVersion(s string) Version {
	var meta string
	var numsStr string
	var pre string

	//v0.0.0.0-pre+meta+asdfa+sdf
	numsStr = strings.TrimPrefix(s, "v")

	parts := strings.Split(s, "+")
	if len(parts) > 1 {
		numsStr = parts[0]                  // 0.0.0.0-pre
		meta = strings.Join(parts[1:], "+") // meta+asdfa+sdf
	}

	//0.0.0.0-pre
	parts = strings.Split(numsStr, "-")
	if len(parts) > 1 {
		numsStr = parts[0] // 0.0.0.0
		pre = parts[1]     // pre
	}

	numSlice := strings.Split(numsStr, ".")
	var ints []int
	for _, n := range numSlice {
		i, _ := strconv.ParseInt(n, 10, 0)
		ints = append(ints, int(i))
	}

	return Version{
		nums:     ints,
		pre:      pre,
		meta:     meta,
		original: s,
	}
}

func (o *Version) Compare(v Version) int {
	r := compareNumArr(0, o.nums, v.nums)
	if r != 0 {
		return r
	}

	//
	if o.pre == "" && v.pre != "" {
		return 1
	} else if o.pre != "" && v.pre == "" {
		return -1
	}

	// metadata is not used to compare

	return 0
}

func compareNumArr(idx int, a, b []int) int {
	aLast := len(a) - 1
	bLast := len(b) - 1

	if idx > aLast && idx > bLast {
		return 0
	}

	var av int = 0
	var bv int = 0

	if idx <= aLast {
		av = a[idx]
	}
	if idx <= bLast {
		bv = b[idx]
	}

	if av > bv {
		return 1
	} else if av < bv {
		return -1
	}
	return compareNumArr(idx+1, a, b)
}

type VersionPattern struct {
	nums     []string
	pre      string
	meta     string
	original string
}

func (o *VersionPattern) Matches(v Version) bool {
	return o.compareNumArr(0, o.nums, v.nums)
}

func (o *VersionPattern) compareNumArr(idx int, a []string, b []int) bool {
	aLast := len(a) - 1
	bLast := len(b) - 1

	if idx > aLast && idx > bLast {
		return true
	}

	var avStr string = "*"
	var av int = 0
	var bv int = 0

	if idx <= aLast {
		avStr = a[idx]
	}
	if idx <= bLast {
		bv = b[idx]
	}

	if avStr == "*" {
		av = bv
	} else {
		tmp, _ := strconv.ParseInt(avStr, 10, 0)
		av = int(tmp)
	}

	if av == bv {
		return o.compareNumArr(idx+1, a, b)
	}

	return false
}

func NewVersionPattern(s string) VersionPattern {
	var meta string
	var numsStr string
	var pre string

	//v0.0.0.0-pre+meta+asdfa+sdf
	numsStr = strings.TrimPrefix(s, "v")

	parts := strings.Split(s, "+")
	if len(parts) > 1 {
		numsStr = parts[0]                  // 0.0.0.0-pre
		meta = strings.Join(parts[1:], "+") // meta+asdfa+sdf
	}

	//0.0.0.0-pre
	parts = strings.Split(numsStr, "-")
	if len(parts) > 1 {
		numsStr = parts[0] // 0.0.0.0
		pre = parts[1]     // pre
	}

	numSlice := strings.Split(numsStr, ".")

	return VersionPattern{nums: numSlice, pre: pre, meta: meta, original: s}
}
