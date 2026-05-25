package words

import (
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
	"github.com/kljensen/snowball/english"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getMessageSlice(srcMessage string) []string {
	srcMessage = strings.ToLower(srcMessage)
	result := strings.FieldsFunc(srcMessage, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return result
}

func removeDuplicates(message []string) []string {
	slices.Sort(message)
	message = slices.Compact(message)
	return message
}

func filterAndStemMessage(message []string, language string) ([]string, error) {
	var result = make([]string, 0, len(message))
	for _, word := range message {
		if english.IsStopWord(word) {
			continue
		}
		stemmed, err := snowball.Stem(word, language, false)
		result = append(result, stemmed)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error while stemming word "+word, "error", err)
		}
	}
	return result, nil
}

func Norm(message string, language string) ([]string, error) {
	var result []string

	result = getMessageSlice(message)

	result, err := filterAndStemMessage(result, language)
	if err != nil {
		slog.Error("Error while filter and stem", "err", err)
		return nil, err
	}
	result = removeDuplicates(result)
	return result, nil
}

func NormLeaveDuplicates(message string, language string) ([]string, error) {
	var result []string

	result = getMessageSlice(message)

	result, err := filterAndStemMessage(result, language)
	if err != nil {
		slog.Error("Error while filter and stem", "err", err)
		return nil, err
	}
	return result, nil
}
