package utils

import (
	"regexp"
)

// Splits a large text string into an array of individual sentences
func SplitBySentence(text string) []string {
	// Regex to find the end of sentences
	re := regexp.MustCompile(`[.!?\n]`)
	// Split the text based on the regex matches
	sentences := re.Split(text, -1)
	// filter out empty strings
	var filteredSentences []string
	for _, sentence := range sentences {
		if len(sentence) > 0 {
			filteredSentences = append(filteredSentences, sentence)
		}
	}
	return filteredSentences
}
