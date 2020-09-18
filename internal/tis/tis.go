package tis

import (
	"errors"
	"regexp"
	"strings"
)

// TIS emulates a TIS-100 program node
type TIS struct {
	Acc int8
	Bak int8
	R1  chan int8
	R2  chan int8
	R3  chan int8
	R4  chan int8

	asm      []string
	labelMap map[string]int
}

// Finds and generates map of labels to index
func generateLabelMap(asm []string) (map[string]int, error) {
	labelRe := regexp.MustCompile("^\\s*(\\w+):")
	labelMap := make(map[string]int)

	for i, line := range asm {
		matches := labelRe.FindStringSubmatch(line)
		if len(matches) == 2 {
			label := strings.ToUpper(matches[1])
			if _, ok := labelMap[label]; ok {
				return nil, errors.New("Cannot repeat label")
			}
			labelMap[label] = i
		}
	}
	return labelMap, nil
}

// New creates new instance of TIS-100 emulator
func New(s string) (*TIS, error) {
	asm := strings.Split(s, "\n")
	labelMap, err := generateLabelMap(asm)

	if err != nil {
		return nil, err
	}

	return &TIS{
		Acc:      0,
		Bak:      0,
		R1:       make(chan int8, 1),
		R2:       make(chan int8, 1),
		R3:       make(chan int8, 1),
		R4:       make(chan int8, 1),
		asm:      asm,
		labelMap: labelMap,
	}, nil
}

// Nop is a no-op
func (t *TIS) Nop() {

}

// Swp swaps ACC and BAK
func (t *TIS) Swp() {
	temp := t.Acc
	t.Acc = t.Bak
	t.Bak = temp
}

// Sav saves ACC to BAK
func (t *TIS) Sav() {
	t.Bak = t.Acc
}

// Neg negates value in Acc
func (t *TIS) Neg() {
	t.Acc = -t.Acc
}

// Jmp jumps to label
func (t *TIS) Jmp(label string) {

}
