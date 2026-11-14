package media

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
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

// buildTestWebP 构造最小 RIFF/WebP 容器：VP8L 图像 chunk + 可选的
// EXIF chunk（奇数长度，验证 pad 处理）与 XMP chunk。
func buildTestWebP(withMetadata bool) []byte {
	chunk := func(fourCC string, payload []byte) []byte {
		head := make([]byte, 8)
		copy(head[0:4], fourCC)
		binary.LittleEndian.PutUint32(head[4:8], uint32(len(payload)))
		out := append(head, payload...)
		if len(payload)%2 == 1 {
			out = append(out, 0) // RIFF 奇数字节 pad
		}
		return out
	}
	body := chunk("VP8L", []byte("\x2f\x00\x00\x00\x00-image-bytes"))
	if withMetadata {
		body = append(body, chunk("EXIF", []byte("Exif-GPS-GPS"))...) // 奇数长度 → 带 pad
		body = append(body, chunk("XMP ", []byte("<x:xmpmeta/>"))...)
	}
	header := make([]byte, 12)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(body)+4))
	copy(header[8:12], "WEBP")
	return append(header, body...)
}

// 防回归（P2-13）：WebP 的 EXIF/XMP chunk 必须被清洗（GPS 载体），图像
// chunk 保留，RIFF 头大小字段改写；无元数据 chunk 时原样返回。
func TestStripWebPMetadataRemovesEXIFAndXMP(t *testing.T) {
	withMeta := buildTestWebP(true)
	cleaned, changed, err := stripImageMetadata(withMeta)
	if err != nil {
		t.Fatalf("strip webp: %v", err)
	}
	if !changed {
		t.Fatal("expected EXIF/XMP chunks to be removed")
	}
	if bytes.Contains(cleaned, []byte("Exif-GPS-GPS")) || bytes.Contains(cleaned, []byte("x:xmpmeta")) {
		t.Fatal("metadata payload still present after scrub")
	}
	if !bytes.Contains(cleaned, []byte("image-bytes")) {
		t.Fatal("image chunk must be preserved")
	}
	if got := binary.LittleEndian.Uint32(cleaned[4:8]); got != uint32(len(cleaned)-8) {
		t.Fatalf("RIFF header size must be rewritten: got %d want %d", got, len(cleaned)-8)
	}

	cleanMeta, changed, err := stripImageMetadata(buildTestWebP(false))
	if err != nil || changed || !bytes.Equal(cleanMeta, buildTestWebP(false)) {
		t.Fatalf("clean webp should pass through: changed=%v err=%v", changed, err)
	}

	// 截断容器（chunk 头越界）→ 解析错误（fail-open 由调用方处理）。
	if _, _, err := stripImageMetadata(withMeta[:len(withMeta)-6]); err == nil {
		t.Fatal("truncated webp should fail parsing")
	}
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

// buildTestJPEGWithFillBytes 在 EXIF APP1 标记前插入 fill 个 0xFF 填充字节
// （fill bytes，ITU-T T.81 B.1.1.2 允许标记前置任意数量 0xFF）。fill=0 时
// 与 buildTestJPEG(true) 逐字节一致。
func buildTestJPEGWithFillBytes(fill int) []byte {
	base := buildTestJPEG(true)
	app1 := bytes.Index(base, []byte{0xFF, 0xE1})
	if app1 < 0 {
		panic("test jpeg missing APP1")
	}
	out := make([]byte, 0, len(base)+fill)
	out = append(out, base[:app1]...)
	out = append(out, bytes.Repeat([]byte{0xFF}, fill)...)
	return append(out, base[app1:]...)
}

// 第 81 轮审计：APP1 前的 0xFF 填充字节是合法 JPEG 结构，不能因此 fail-open。
// 清洗后 GPS EXIF 必须消失，输出与「无 EXIF 版本」逐字节一致（填充随被删
// 段一并移除，保留段前的填充则原样保留，见下一个用例）。
func TestStripJPEGMetadataHandlesFillBytes(t *testing.T) {
	entropy := []byte{0x12, 0x34, 0xFF, 0xD0, 0x56, 0x78, 0xFF, 0x00, 0x9A}
	for _, fill := range []int{0, 1, 2, 3, 8} {
		t.Run(fmt.Sprintf("fill=%d", fill), func(t *testing.T) {
			src := buildTestJPEGWithFillBytes(fill)
			cleaned, changed, err := stripImageMetadata(src)
			if err != nil {
				t.Fatalf("fill bytes must not fail parsing: %v", err)
			}
			if !changed {
				t.Fatal("expected EXIF APP1 to be removed")
			}
			if bytes.Contains(cleaned, []byte("GPS-GPS-GPS")) {
				t.Fatal("EXIF payload still present after scrub")
			}
			if !bytes.Contains(cleaned, []byte("JFIF")) {
				t.Fatal("JFIF APP0 must be preserved")
			}
			if !bytes.Contains(cleaned, entropy) {
				t.Fatal("entropy data must be preserved verbatim")
			}
			if !bytes.Equal(cleaned, buildTestJPEG(false)) {
				t.Fatalf("cleaned jpeg must equal the version without APP1 (before=% x after=% x)", src, cleaned)
			}
		})
	}
}

// 保留段（DQT 等）前的填充字节属于文件字节流，清洗只能删元数据，不能顺带
// 改动/丢掉保留段前的填充（否则输出与「仅去掉 APP1」的期望不一致）。
func TestStripJPEGMetadataPreservesFillBytesBeforeKeptSegments(t *testing.T) {
	fills := bytes.Repeat([]byte{0xFF}, 3)
	splice := func(base []byte, at int, ins []byte) []byte {
		out := make([]byte, 0, len(base)+len(ins))
		out = append(out, base[:at]...)
		out = append(out, ins...)
		return append(out, base[at:]...)
	}

	base := buildTestJPEG(true)
	dqt := bytes.Index(base, []byte{0xFF, 0xDB})
	if dqt < 0 {
		t.Fatal("test jpeg missing DQT")
	}
	src := splice(base, dqt, fills)

	cleaned, changed, err := stripImageMetadata(src)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if !changed {
		t.Fatal("expected EXIF APP1 to be removed")
	}
	if bytes.Contains(cleaned, []byte("GPS-GPS-GPS")) {
		t.Fatal("EXIF payload still present after scrub")
	}
	noEXIF := buildTestJPEG(false)
	noEXIFDQT := bytes.Index(noEXIF, []byte{0xFF, 0xDB})
	if noEXIFDQT < 0 {
		t.Fatal("test jpeg without exif missing DQT")
	}
	if want := splice(noEXIF, noEXIFDQT, fills); !bytes.Equal(cleaned, want) {
		t.Fatalf("fill bytes before kept segment must be preserved verbatim (got=% x want=% x)", cleaned, want)
	}
}

// pngChunkWithCRC 构造带真实 CRC 的 PNG 块（CRC 覆盖 type+data，PNG 3rd ed. 5.3）。
func pngChunkWithCRC(chunkType string, payload []byte) []byte {
	chunk := make([]byte, 8, 12+len(payload))
	binary.BigEndian.PutUint32(chunk[0:4], uint32(len(payload)))
	copy(chunk[4:8], chunkType)
	chunk = append(chunk, payload...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
}

// buildTestPNGChunks 用标准库编码一张真实可解码的 4x4 PNG，再把 extraChunks
// 插到 IEND 之前。用于验证清洗后图像仍可解码、块结构与 CRC 合法（只看
// payload 是否消失的断言漏不掉「删块后字节错位」）。
func buildTestPNGChunks(tb testing.TB, extraChunks ...[]byte) []byte {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 0x20, 0x40, 0x80, 0xFF
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		tb.Fatalf("encode png: %v", err)
	}
	encoded := buf.Bytes()
	if len(encoded) < 12 || string(encoded[len(encoded)-8:len(encoded)-4]) != "IEND" {
		tb.Fatalf("encoded png missing IEND tail: % x", encoded)
	}
	out := append([]byte(nil), encoded[:len(encoded)-12]...)
	for _, chunk := range extraChunks {
		out = append(out, chunk...)
	}
	return append(out, encoded[len(encoded)-12:]...)
}

// pngTextPayload 返回 tEXt 块数据：keyword + NUL + Latin-1 文本。
func pngTextPayload(keyword, text string) []byte {
	return append(append([]byte(keyword), 0), []byte(text)...)
}

// pngCompressedTextPayload 返回 zTXt 块数据：keyword + NUL + 压缩方法(0) +
// zlib 压缩文本。
func pngCompressedTextPayload(tb testing.TB, keyword, text string) []byte {
	tb.Helper()
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write([]byte(text)); err != nil {
		tb.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		tb.Fatalf("zlib close: %v", err)
	}
	payload := append(append([]byte(keyword), 0), 0) // keyword + NUL + 压缩方法 0
	return append(payload, compressed.Bytes()...)
}

// pngInternationalTextPayload 返回未压缩的 iTXt 块数据：keyword + NUL +
// 压缩标志(0) + 压缩方法(0) + 语言标签 + NUL + 翻译 keyword + NUL + UTF-8 文本。
func pngInternationalTextPayload(keyword, text string) []byte {
	payload := append(append([]byte(keyword), 0), 0, 0)
	payload = append(payload, 0) // 语言标签为空
	payload = append(payload, 0) // 翻译 keyword 为空
	return append(payload, []byte(text)...)
}

// assertPNGChunkCRCs 遍历 PNG 块，校验长度边界与 CRC，并要求 IEND 是最后一块
// （无尾随字节）。
func assertPNGChunkCRCs(t *testing.T, data []byte) {
	t.Helper()
	if !bytes.HasPrefix(data, pngSignature) {
		t.Fatalf("missing png signature: % x", data)
	}
	for pos := 8; pos < len(data); {
		if pos+8 > len(data) {
			t.Fatalf("truncated png chunk header at %d", pos)
		}
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		end := pos + 12 + length
		if end > len(data) {
			t.Fatalf("png chunk %q length %d overruns buffer", data[pos+4:pos+8], length)
		}
		chunkType := string(data[pos+4 : pos+8])
		want := binary.BigEndian.Uint32(data[pos+8+length : end])
		if got := crc32.ChecksumIEEE(data[pos+4 : pos+8+length]); got != want {
			t.Fatalf("png chunk %q crc mismatch: got %#x want %#x", chunkType, got, want)
		}
		if chunkType == "IEND" {
			if end != len(data) {
				t.Fatalf("trailing bytes after IEND: %d", len(data)-end)
			}
			return
		}
		pos = end
	}
	t.Fatal("png has no IEND chunk")
}

// assertNoPNGMetadataChunks 校验清洗结果里不再有任何 EXIF/XMP 元数据块。
func assertNoPNGMetadataChunks(t *testing.T, data []byte) {
	t.Helper()
	for pos := 8; pos+8 <= len(data); {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		end := pos + 12 + length
		if end > len(data) {
			t.Fatalf("png chunk %q length %d overruns buffer", data[pos+4:pos+8], length)
		}
		chunkType := string(data[pos+4 : pos+8])
		if pngChunkCarriesMetadata(chunkType, data[pos+8:pos+8+length]) {
			t.Fatalf("metadata chunk %q still present after scrub", chunkType)
		}
		pos = end
	}
}

// 第 81 轮审计：PNG 此前只删 eXIf，XMP（tEXt/zTXt/iTXt）与 ImageMagick 的
// 文本版 EXIF 仍随图发布。清洗后这些块必须全部删除，非元数据文本块保留，
// 且图像仍可被标准解码器解码、保留块 CRC 全部正确。
func TestStripPNGMetadataRemovesXMPAndTextEXIF(t *testing.T) {
	const xmpMarker = "GPS-LOCATION"
	const xmpXML = "<x:xmpmeta>" + xmpMarker + "</x:xmpmeta>"
	src := buildTestPNGChunks(t,
		pngChunkWithCRC("eXIf", []byte("MM\x00*"+xmpMarker)),
		pngChunkWithCRC("tEXt", pngTextPayload("XML:com.adobe.xmp", xmpXML)),
		pngChunkWithCRC("iTXt", pngInternationalTextPayload("XML:com.adobe.xmp", xmpXML)),
		pngChunkWithCRC("zTXt", pngCompressedTextPayload(t, "XML:com.adobe.xmp", xmpXML)),
		pngChunkWithCRC("tEXt", pngTextPayload("Raw profile type exif", "4558494600"+xmpMarker)),
		pngChunkWithCRC("tEXt", pngTextPayload("Raw profile type xmp", xmpMarker)),
		pngChunkWithCRC("tEXt", pngTextPayload("Comment", "kept-comment")),
	)

	cleaned, changed, err := stripImageMetadata(src)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}
	if !changed {
		t.Fatal("expected EXIF/XMP chunks to be removed")
	}
	if bytes.Contains(cleaned, []byte(xmpMarker)) {
		t.Fatal("metadata payload still present after scrub")
	}
	if !bytes.Contains(cleaned, []byte("kept-comment")) {
		t.Fatal("non-metadata tEXt must be preserved")
	}
	assertNoPNGMetadataChunks(t, cleaned)
	assertPNGChunkCRCs(t, cleaned)

	img, err := png.Decode(bytes.NewReader(cleaned))
	if err != nil {
		t.Fatalf("cleaned png must still decode: %v", err)
	}
	if got := img.Bounds(); got.Dx() != 4 || got.Dy() != 4 {
		t.Fatalf("decoded png bounds = %v, want 4x4", got)
	}
}

// 非元数据文本块 / 无 XMP 的 PNG：清洗应逐字节不变（幂等），图像仍可解码；
// 无 NUL 分隔符的畸形文本块不 panic、不按元数据块误删。
func TestStripPNGMetadataKeepsNonMetadataText(t *testing.T) {
	base := buildTestPNGChunks(t, pngChunkWithCRC("tEXt", pngTextPayload("Comment", "kept-comment")))
	cleaned, changed, err := stripImageMetadata(base)
	if err != nil || changed || !bytes.Equal(cleaned, base) {
		t.Fatalf("png without metadata must be byte-identical: changed=%v err=%v", changed, err)
	}
	assertPNGChunkCRCs(t, cleaned)
	if _, err := png.Decode(bytes.NewReader(cleaned)); err != nil {
		t.Fatalf("clean png must decode: %v", err)
	}

	malformedText := buildTestPNGChunks(t, pngChunkWithCRC("tEXt", []byte("no-nul-separator")))
	cleaned, changed, err = stripImageMetadata(malformedText)
	if err != nil || changed || !bytes.Equal(cleaned, malformedText) {
		t.Fatalf("text chunk without keyword separator must pass through: changed=%v err=%v", changed, err)
	}
}
