package media

import (
	"bytes"
	"testing"
)

// JPEG 分段解析的错误分支：段头越界、非 0xFF 标记、SOS 段头越界必须返回
// 解析错误（调用方 fail-open 保留原文件）。
func TestStripJPEGMetadataRejectsMalformedSegments(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"segment header beyond buffer", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}},
		{"non-marker byte after segment", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x02, 0x12}},
		{"truncated marker", []byte{0xFF, 0xD8, 0xFF}},
		{"sos segment beyond buffer", []byte{0xFF, 0xD8, 0xFF, 0xDA, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := stripImageMetadata(tc.data); err == nil {
				t.Fatalf("malformed jpeg %v must fail parsing", tc.data)
			}
		})
	}
}

// 无 SOS 的 JPEG（SOI + EOI，或恰好在段边界结束）按「无更多可剥离段」原样返回。
func TestStripJPEGMetadataHandlesEOIAndSegmentEnd(t *testing.T) {
	for name, src := range map[string][]byte{
		"eoi immediately after soi": {0xFF, 0xD8, 0xFF, 0xD9},
		"ends at segment boundary":  {0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x02},
		"app0 then eoi without sos": {0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x02, 0xFF, 0xD9},
	} {
		t.Run(name, func(t *testing.T) {
			cleaned, changed, err := stripImageMetadata(src)
			if err != nil || changed || !bytes.Equal(cleaned, src) {
				t.Fatalf("jpeg without metadata: cleaned=%v changed=%v err=%v", cleaned, changed, err)
			}
		})
	}
}

// PNG 块解析错误：块长度越界、尾部残留字节。
func TestStripPNGMetadataRejectsMalformedChunks(t *testing.T) {
	overflow := append(append([]byte(nil), pngSignature...), 0xFF, 0xFF, 0xFF, 0xFF, 'I', 'H', 'D', 'R', 0, 0, 0, 0)
	if _, _, err := stripImageMetadata(overflow); err == nil {
		t.Fatal("chunk length beyond buffer must fail parsing")
	}

	trailing := append(append([]byte(nil), pngSignature...), 1, 2, 3, 4)
	if _, _, err := stripImageMetadata(trailing); err == nil {
		t.Fatal("png with trailing bytes must fail parsing")
	}
}

// WebP 容器尾部残留字节（RIFF chunk 遍历未对齐到文件末尾）必须报错。
func TestStripWebPMetadataRejectsTrailingBytes(t *testing.T) {
	data := []byte("RIFF\x00\x00\x00\x00WEBP" + "VP8L\x00\x00\x00\x00" + "\x01\x02\x03")
	if _, _, err := stripImageMetadata(data); err == nil {
		t.Fatal("webp with trailing bytes must fail parsing")
	}
}
