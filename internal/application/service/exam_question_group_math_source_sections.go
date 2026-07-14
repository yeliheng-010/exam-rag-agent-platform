package service

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	mathOuterQuestionPattern = regexp.MustCompile(`[（(](\d+)[）)]`)
	mathInnerQuestionPattern = regexp.MustCompile(`[（(](i{1,3}|iv|v|ⅰ|ⅱ|ⅲ)[）)]`)
)

type mathSourceSection struct {
	QuestionNo string
	Stem       string
}

func splitMathSubjectiveSections(content string, number int) (string, []mathSourceSection) {
	outer := mathOuterQuestionPattern.FindAllStringSubmatchIndex(content, -1)
	if len(outer) == 0 {
		return "", nil
	}
	material := strings.TrimSpace(content[:outer[0][0]])
	sections := make([]mathSourceSection, 0, len(outer))
	for index, match := range outer {
		end := len(content)
		if index+1 < len(outer) {
			end = outer[index+1][0]
		}
		outerNo := content[match[2]:match[3]]
		sectionText := strings.TrimSpace(content[match[1]:end])
		inner := splitMathInnerSections(sectionText, number, outerNo)
		if len(inner) > 0 {
			sections = append(sections, inner...)
		} else {
			sections = append(sections, mathSourceSection{QuestionNo: numberWithSuffix(number, outerNo), Stem: sectionText})
		}
	}
	return material, sections
}

func splitMathInnerSections(content string, number int, outerNo string) []mathSourceSection {
	inner := mathInnerQuestionPattern.FindAllStringSubmatchIndex(content, -1)
	if len(inner) == 0 {
		return nil
	}
	prefix := strings.TrimSpace(content[:inner[0][0]])
	sections := make([]mathSourceSection, 0, len(inner))
	for index, match := range inner {
		end := len(content)
		if index+1 < len(inner) {
			end = inner[index+1][0]
		}
		roman := normalizeMathRoman(content[match[2]:match[3]])
		stem := strings.TrimSpace(prefix + "\n" + strings.TrimSpace(content[match[1]:end]))
		sections = append(sections, mathSourceSection{
			QuestionNo: numberWithSuffix(number, outerNo) + "(" + roman + ")", Stem: stem,
		})
	}
	return sections
}

func numberWithSuffix(number int, suffix string) string {
	return strconv.Itoa(number) + "(" + strings.TrimSpace(suffix) + ")"
}

func normalizeMathRoman(value string) string {
	replacer := strings.NewReplacer("ⅰ", "i", "ⅱ", "ii", "ⅲ", "iii")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}
