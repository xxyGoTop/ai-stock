package provider

import (
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// decodeMaybeGBK 将腾讯等 GBK 行情正文转为 UTF-8。
// 若本身已是合法 UTF-8（且不含大量替换符），则原样返回。
func decodeMaybeGBK(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	if utf8.Valid(raw) && !looksLikeMojibake(raw) {
		return string(raw)
	}
	reader := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.GBK.NewDecoder())
	out, err := io.ReadAll(reader)
	if err != nil || len(out) == 0 {
		return string(raw)
	}
	return string(out)
}

func looksLikeMojibake(raw []byte) bool {
	// 腾讯 GBK 中文名常见高位字节；若按 UTF-8 读会大量无效或替换
	invalid := 0
	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRune(raw[i:])
		if r == utf8.RuneError && size == 1 {
			invalid++
		}
		i += size
	}
	return invalid >= 2
}
