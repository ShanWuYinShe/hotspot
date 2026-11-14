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
		{"fill bytes then eof", []byte{0xFF, 0xD8, 0xFF, 0xFF, 0xFF}},
		{"standalone rst marker in header", []byte{0xFF, 0xD8, 0xFF, 0xD0}},
		{"stuffed zero byte in header", []byte{0xFF, 0xD8, 0xFF, 0x00}},
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
		"fill bytes before eoi":     {0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x02, 0xFF, 0xFF, 0xFF, 0xD9},
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

// FuzzStripImageMetadata：上传内容完全不可信，图片解析历史上出现过 panic。
// 契约：任意字节输入不 panic；失败返回 nil 字节（调用方 fail-open 保留原图）；
// 成功时输出不长于输入，「未修改」必须逐字节一致，「已修改」必须幂等
// （第 81 轮审计要求媒体解析器有 fuzz 覆盖，禁止静默改写文件）。
func FuzzStripImageMetadata(f *testing.F) {
	pngSeed := buildTestPNGChunks(f,
		pngChunkWithCRC("tEXt", pngTextPayload("XML:com.adobe.xmp", "<x:xmpmeta>GPS</x:xmpmeta>")),
		pngChunkWithCRC("eXIf", []byte("Exif\x00\x00GPS")),
	)
	seeds := [][]byte{
		buildTestJPEG(true),
		buildTestJPEG(false),
		buildTestJPEGWithFillBytes(2),
		buildTestWebP(true),
		buildTestWebP(false),
		pngSeed,
		{0xFF, 0xD8, 0xFF, 0xE1, 0x7F, 0xFF, 0x01, 0x02},                     // JPEG 段长越界
		{0xFF, 0xD8, 0xFF, 0xFF, 0xFF},                                       // 纯填充后截断
		append(append([]byte(nil), pngSignature...), 0xFF, 0xFF, 0xFF, 0xFF), // PNG 块长越界
		append(append([]byte(nil), pngSignature...), 1, 2, 3, 4),             // PNG 尾随残余字节
		[]byte("RIFF\x00\x00\x00\x00WEBP" + "VP8L\x00\x00\x00\x00"),          // 空 WebP body
		[]byte("%PDF-1.7 not an image"),
		[]byte(nil),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		cleaned, changed, err := stripImageMetadata(data)
		if err != nil {
			if cleaned != nil {
				t.Fatalf("error must not return bytes: %v", err)
			}
			return
		}
		if len(cleaned) > len(data) {
			t.Fatalf("scrub must not grow data: before=%d after=%d", len(data), len(cleaned))
		}
		if !changed {
			if !bytes.Equal(cleaned, data) {
				t.Fatalf("unchanged scrub must be byte-identical: before=%x after=%x", data, cleaned)
			}
			return
		}
		if again, changedAgain, err := stripImageMetadata(cleaned); err != nil || changedAgain || !bytes.Equal(again, cleaned) {
			t.Fatalf("scrub must be idempotent: err=%v changed=%v", err, changedAgain)
		}
	})
}
