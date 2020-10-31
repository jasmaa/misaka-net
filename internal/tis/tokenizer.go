package tis

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// GenerateLabelMap maps defined labels to instruction location
func GenerateLabelMap(instrArr []string) (map[string]int, error) {
	labelRe := regexp.MustCompile(`^\s*(\w+):`)
	labelMap := make(map[string]int)

	for i, line := range instrArr {
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

// Tokenize converts each instruction string into a string array of tokens
func Tokenize(instrArr []string, labelMap map[string]int) ([][]string, error) {
	// Map instructions
	asm := make([][]string, len(instrArr))
	for i, instr := range instrArr {
		// Get rid of labels and whitespace
		prefixRe := regexp.MustCompile(`^(\s*\w+:)?\s*`)
		if indices := prefixRe.FindStringIndex(instr); indices != nil {
			end := indices[1]
			instr = instr[end:]
		}

		// Convert instr strings to tokens
		if len(instr) == 0 {
			// <Label>:
			asm[i] = []string{"NOP"}
		} else if m := regexp.MustCompile(`^#.*$`).FindStringSubmatch(instr); len(m) > 0 {
			// #<Comment>
			asm[i] = []string{"NOP"}
		} else if m := regexp.MustCompile(`^(NOP|SWP|SAV|NEG)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// NOP|SWP|SAV|NEG
			asm[i] = []string{m[1]}
		} else if m := regexp.MustCompile(`^MOV\s+(-?\d+)\s*,\s+(ACC|NIL)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <VAL>, <DST>
			asm[i] = []string{"MOV_VAL_LOCAL", m[1], m[2]}
		} else if m := regexp.MustCompile(`^MOV\s+(-?\d+)\s*,\s+(\w+:R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <VAL>, <DST>
			asm[i] = []string{"MOV_VAL_NETWORK", m[1], m[2]}
		} else if m := regexp.MustCompile(`^MOV\s+(ACC|NIL|R[0123])\s*,\s+(ACC|NIL)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <SRC>, <DST>
			asm[i] = []string{"MOV_SRC_LOCAL", m[1], m[2]}
		} else if m := regexp.MustCompile(`^MOV\s+(ACC|NIL|R[0123])\s*,\s+(\w+:R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <SRC>, <DST>
			asm[i] = []string{"MOV_SRC_NETWORK", m[1], m[2]}
		} else if m := regexp.MustCompile(`^(ADD|SUB)\s+(-?\d+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// ADD|SUB <VAL>
			asm[i] = []string{fmt.Sprintf("%s_VAL", m[1]), m[2]}
		} else if m := regexp.MustCompile(`^(ADD|SUB)\s+(ACC|NIL|R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// ADD|SUB <SRC>
			asm[i] = []string{fmt.Sprintf("%s_SRC", m[1]), m[2]}
		} else if m := regexp.MustCompile(`^(JMP|JEZ|JNZ|JGZ|JLZ)\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JMP|JEZ|JNZ|JGZ|JLZ <LABEL>
			label := strings.ToUpper(m[2])
			if _, ok := labelMap[label]; ok {
				asm[i] = []string{m[1], label}
			} else {
				return nil, fmt.Errorf("line %v, label '%s' was not declared", i, label)
			}
		} else if m := regexp.MustCompile(`^JRO\s+(-?\d+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JRO <VAL>
			asm[i] = []string{"JRO_VAL", m[1]}
		} else if m := regexp.MustCompile(`^JRO\s+(ACC|NIL|R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JRO <SRC>
			asm[i] = []string{"JRO_SRC", m[1]}
		} else if m := regexp.MustCompile(`^PUSH\s+(-?\d+)\s*,\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// PUSH <VAL>, <DST>
			asm[i] = []string{"PUSH_VAL", m[1], m[2]}
		} else if m := regexp.MustCompile(`^PUSH\s+(ACC|NIL|R[0123])\s*,\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// PUSH <SRC>, <DST>
			asm[i] = []string{"PUSH_SRC", m[1], m[2]}
		} else if m := regexp.MustCompile(`^POP\s+(\w+)\s*,\s+(ACC|NIL)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// POP <SRC>, <DST>
			asm[i] = []string{"POP", m[1], m[2]}
		} else if m := regexp.MustCompile(`^IN\s+(ACC|NIL)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// IN <DST>
			asm[i] = []string{"IN", m[1]}
		} else if m := regexp.MustCompile(`^OUT\s+(-?\d+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// OUT <VAL>
			asm[i] = []string{"OUT_VAL", m[1]}
		} else if m := regexp.MustCompile(`^OUT\s+(ACC|NIL|R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// OUT <SRC>
			asm[i] = []string{"OUT_SRC", m[1]}
		} else {
			return nil, fmt.Errorf("line %v, '%s' not a valid instruction", i, instr)
		}
	}

	return asm, nil
}
