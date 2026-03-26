package mapper

import (
	"strings"

	"github.com/mozillazg/go-pinyin"
)

var pinyinArgs = func() pinyin.Args {
	args := pinyin.NewArgs()
	args.Style = pinyin.Tone
	return args
}()

func ConvertToPinyin(hanzi string) string {
	result := pinyin.Pinyin(hanzi, pinyinArgs)
	parts := make([]string, 0, len(result))
	for _, syllable := range result {
		if len(syllable) > 0 {
			parts = append(parts, syllable[0])
		}
	}
	return strings.Join(parts, " ")
}
