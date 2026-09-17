package media

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildTestJPEG 构造最小合法 JPEG 结构：SOI + APP0(JFIF) + [APP1(EXIF)] +
// DQT + SOF0 + SOS + 熵数据(含 RSTn 填充形态) + EOI。
func buildTestJPEG(withEXIF bool) []byte {
	segment := func(marker byte, payload []byte) []byte {
		out := []byte{0xFF, marker}
		length := make([]byte, 2)
		binary.BigEndian.PutUint16(length, uint16(len(payload)+2))
		out = append(out, length...)
		return append(out, payload...)
	}
	out := []byte{0xFF, 0xD8}
	out = append(out, segment(0xE0, []byte("JFIF\x00\x01\x02\x00\x00\x01\x00\x01"))...)
	if withEXIF {
		// APP1/EXIF：真实 EXIF 头 + 一段模拟 GPS IFD 载荷。
		out = append(out, segment(0xE1, []byte("Exif\x00\x00"+"\x4d\x4d\x00\x2aGPS-GPS-GPS"))...)
	}
	out = append(out, segment(0xDB, []byte("\x00\x01\x02\x03"))...)         // DQT
	out = append(out, segment(0xC0, []byte("\x08\x00\x01\x00\x01\x01"))...) // SOF0
	out = append(out, segment(0xDA, []byte("\x01\x01\x00\x00\x3f\x00"))...) // SOS
	out = append(out, 0x12, 0x34, 0xFF, 0xD0, 0x56, 0x78, 0xFF, 0x00, 0x9A) // 熵数据（含 RSTn 与 FF00 填充）
	out = append(out, 0xFF, 0xD9)                                           // EOI
	return out
}

func TestStripJPEGMetadataRemovesEXIFKeepsStructure(t *testing.T) {
	withEXIF := buildTestJPEG(true)
	cleaned, changed, err := stripImageMetadata(withEXIF)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if !changed {
		t.Fatal("expected EXIF APP1 to be removed")
	}
	// EXIF 载荷消失。
	if bytes.Contains(cleaned, []byte("GPS-GPS-GPS")) {
		t.Fatal("EXIF payload still present after scrub")
	}
	// JFIF APP0 保留。
	if !bytes.Contains(cleaned, []byte("JFIF")) {
		t.Fatal("JFIF APP0 must be preserved")
	}
	// 熵编码数据（含 RSTn/FF00 填充）原样保留。
	if !bytes.Contains(cleaned, []byte{0x12, 0x34, 0xFF, 0xD0, 0x56, 0x78, 0xFF, 0x00, 0x9A}) {
		t.Fatal("entropy data (incl. RSTn/FF00) must be preserved verbatim")
	}
	// 结构自检：仍是合法 SOI 开头、EOI 结尾。
	if cleaned[0] != 0xFF || cleaned[1] != 0xD8 || cleaned[len(cleaned)-2] != 0xFF || cleaned[len(cleaned)-1] != 0xD9 {
		t.Fatal("cleaned jpeg must start with SOI and end with EOI")
	}
	// 无元数据版本必须逐字节不变（幂等且无重编码损失）。
	noEXIF := buildTestJPEG(false)
	if !bytes.Equal(cleaned, noEXIF) {
		t.Fatal("cleaned jpeg should equal the version without APP1")
	}
}

func TestStripJPEGMetadataNoopWhenClean(t *testing.T) {
	src := buildTestJPEG(false)
	cleaned, changed, err := stripImageMetadata(src)
	if err != nil || changed {
		t.Fatalf("expected noop: changed=%v err=%v", changed, err)
	}
	if !bytes.Equal(cleaned, src) {
		t.Fatal("clean jpeg must be byte-identical")
	}
}

func pngChunk(chunkType string, data []byte) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], chunkType)
	out = append(out, data...)
	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, 0xDEADBEEF)
	return append(out, crc...)
}

func TestStripPNGMetadataRemovesEXIFChunk(t *testing.T) {
	src := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	src = append(src, pngChunk("IHDR", []byte("0000000d"))...)
	src = append(src, pngChunk("eXIf", []byte("MM\x00\x2aGPS-LOCATION"))...)
	src = append(src, pngChunk("IDAT", []byte("imagedata"))...)
	src = append(src, pngChunk("IEND", nil)...)

	cleaned, changed, err := stripImageMetadata(src)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if !changed {
		t.Fatal("expected eXIf chunk to be removed")
	}
	if bytes.Contains(cleaned, []byte("GPS-LOCATION")) {
		t.Fatal("eXIf payload still present after scrub")
	}
	if !bytes.Contains(cleaned, []byte("imagedata")) {
		t.Fatal("IDAT must be preserved")
	}
	if !bytes.HasSuffix(cleaned, pngChunk("IEND", nil)) {
		t.Fatal("IEND must be the final chunk")
	}
}

func TestStripImageMetadataPassthroughNonImages(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("%PDF-1.7 fake pdf"),
		[]byte("RIFF\x00\x00\x00\x00WAVEfmt "),
		nil,
	} {
		cleaned, changed, err := stripImageMetadata(data)
		if err != nil || changed || !bytes.Equal(cleaned, data) {
			t.Fatalf("expected passthrough: changed=%v err=%v", changed, err)
		}
	}
}

func TestStripJPEGMetadataRejectsMalformed(t *testing.T) {
	// 段长度字段声称超出缓冲。
	broken := []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x7F, 0xFF, 0x01, 0x02}
	if _, _, err := stripImageMetadata(broken); err == nil {
		t.Fatal("expected malformed jpeg error")
	}
}
