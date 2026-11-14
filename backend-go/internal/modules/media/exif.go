package media

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// EXIF/元数据清洗（成熟度审计：带 GPS 的照片原图此前原样分发，位置社交
// 产品下可暴露发布坐标之外的住址级位置）。纯 Go 段/块解析，无外部依赖。
//
// 策略：JPEG 删除全部 APPn 段（APP1=EXIF/XMP、APP13=Photoshop IPTC），
// 保留 JFIF 所需的 APP0；标记前的 0xFF 填充字节（fill bytes，ITU-T T.81
// B.1.1.2）属合法结构，跳过继续解析；SOS 之后是熵编码数据（内含 0xFFxx
// 填充与 RSTn 重启标记，不可按段解析），整段原样拷贝到 EOI。PNG 删除
// eXIf 与 XMP/文本版 EXIF 块（iTXt/zTXt/tEXt 的 "XML:com.adobe.xmp"、
// "Raw profile type exif"、"Raw profile type xmp"）。
// WebP（RIFF 容器）删除 EXIF 与 XMP chunk（P2-13：此前走 default 原样
// 放行，WebP 的 EXIF chunk 同样携带 GPS，魔数校验放行 webp 而清洗跳过
// 属覆盖缺口）。其余格式原样返回。解析失败返回错误——调用方按
// fail-open 处理（保留原文件并告警），不能因清洗失败阻塞合法上传。

var errMalformedJPEG = errors.New("exif: malformed jpeg")
var errMalformedPNG = errors.New("exif: malformed png")
var errMalformedWebP = errors.New("exif: malformed webp")

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}

// stripImageMetadata 返回清洗后的字节与是否发生了修改。
func stripImageMetadata(data []byte) ([]byte, bool, error) {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return stripJPEGMetadata(data)
	case len(data) >= 8 && bytes.Equal(data[:8], pngSignature):
		return stripPNGMetadata(data)
	case isWebP(data):
		return stripWebPMetadata(data)
	default:
		return data, false, nil
	}
}

// stripJPEGMetadata 遍历 JPEG 段并删除 APP1~APPn（EXIF/XMP/IPTC/厂商注释），
// 保留 APP0（JFIF 基本参数，部分解码器依赖且不含元数据载荷）。
//
// 标记前的 0xFF 填充字节（fill bytes）是合法结构：JPEG 标记允许前置任意数量
// 0xFF，标记本体是其后第一个非 0xFF 字节。此前把连续 0xFF 当作畸形段直接
// 返回解析错误，调用方 fail-open 保留原图——攻击者只需在 APP1 前插入一个
// 0xFF 填充，带 GPS 的 EXIF 就原样发布（第 81 轮审计）。现在严格按段结构
// 跳过填充继续解析；只有真正无长度字段的独立标记（RSTn/TEM/SOI 出现在
// SOS 前的段流里）或段头/长度越界才报错（保留原图并告警）。
func stripJPEGMetadata(data []byte) ([]byte, bool, error) {
	// segmentEnd 返回标记起始位置 markerPos 所在段的结束下标。段长字段位于
	// 标记之后 2 字节，长度值含其自身 2 字节，故 end = markerPos+1+length。
	segmentEnd := func(markerPos int) (int, error) {
		if markerPos+3 > len(data) {
			return 0, errMalformedJPEG
		}
		length := int(binary.BigEndian.Uint16(data[markerPos+1 : markerPos+3]))
		end := markerPos + 1 + length
		if length < 2 || end > len(data) {
			return 0, errMalformedJPEG
		}
		return end, nil
	}

	var out bytes.Buffer
	out.Grow(len(data))
	out.Write(data[:2]) // SOI
	pos := 2
	changed := false
	for pos < len(data) {
		if data[pos] != 0xFF {
			return nil, false, errMalformedJPEG
		}
		// 跳过标记前的 0xFF 填充字节，markerPos 指向标记本体。
		markerPos := pos
		for markerPos < len(data) && data[markerPos] == 0xFF {
			markerPos++
		}
		if markerPos >= len(data) {
			return nil, false, errMalformedJPEG
		}
		marker := data[markerPos]
		switch marker {
		case 0xD9: // EOI
			out.Write(data[pos:])
			return out.Bytes(), changed, nil
		case 0xDA: // SOS：其后为熵编码数据 + EOI，无更多可剥离段
			end, err := segmentEnd(markerPos)
			if err != nil {
				return nil, false, err
			}
			out.Write(data[pos:end])
			out.Write(data[end:])
			return out.Bytes(), changed, nil
		}
		// 无长度字段的独立标记（TEM/SOI/RSTn 与熵数据专用的 0x00 填充字节）
		// 本不应出现在 SOS 前的段流里：真畸形输入，保留原图并告警。
		if marker == 0x00 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0xD8 {
			return nil, false, errMalformedJPEG
		}
		end, err := segmentEnd(markerPos)
		if err != nil {
			return nil, false, err
		}
		if marker >= 0xE1 && marker <= 0xEF {
			changed = true // 丢弃 APP1~APP15（EXIF/XMP/IPTC/厂商注释）；其前置填充一并丢弃
		} else {
			out.Write(data[pos:end]) // 保留段连同其前置填充字节原样拷贝
		}
		pos = end
	}
	return out.Bytes(), changed, nil
}

// stripPNGMetadata 删除 PNG 的 EXIF/XMP 元数据块（GPS/隐私载体），其余块
// 原样保留（长度/CRC 随块整体拷贝，不需重算）。
//
// 覆盖三类载体（第 81 轮审计：此前只删 eXIf，XMP 随 iTXt 一并发布）：
//   - eXIf：PNG 1.5+ 的标准 EXIF chunk；
//   - "XML:com.adobe.xmp"：XMP 规范写入 tEXt/zTXt/iTXt 的 keyword；
//   - "Raw profile type exif" / "Raw profile type xmp"：ImageMagick 把原始
//     EXIF/XMP 档案转成十六进制写进 tEXt/zTXt 的形态。
//
// 文本块结构为 keyword + NUL 分隔符 + 其余字段（PNG 3rd ed. 11.3.4.2/4/5），
// 取首个 NUL 前的 keyword 精确比较；非元数据 keyword 的文本块保留。
func stripPNGMetadata(data []byte) ([]byte, bool, error) {
	var out bytes.Buffer
	out.Grow(len(data))
	out.Write(data[:8]) // 签名
	pos := 8
	changed := false
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		chunkEnd := pos + 12 + length // len(4) + type(4) + data + crc(4)
		if chunkEnd > len(data) {
			return nil, false, errMalformedPNG
		}
		chunkType := string(data[pos+4 : pos+8])
		if pngChunkCarriesMetadata(chunkType, data[pos+8:pos+8+length]) {
			changed = true
		} else {
			out.Write(data[pos:chunkEnd])
		}
		pos = chunkEnd
	}
	if pos != len(data) {
		return nil, false, errMalformedPNG
	}
	return out.Bytes(), changed, nil
}

// pngChunkCarriesMetadata 判定 PNG chunk 是否为 EXIF/XMP 元数据载体。
func pngChunkCarriesMetadata(chunkType string, payload []byte) bool {
	if chunkType == "eXIf" {
		return true
	}
	switch chunkType {
	case "tEXt", "zTXt", "iTXt":
	default:
		return false
	}
	nul := bytes.IndexByte(payload, 0)
	if nul < 0 {
		return false // 无 keyword 分隔符的畸形文本块：不按元数据块处理（不猜测）
	}
	switch string(payload[:nul]) {
	case "XML:com.adobe.xmp", "Raw profile type exif", "Raw profile type xmp":
		return true
	}
	return false
}

// isWebP 判定 RIFF 容器 + WEBP 标识（与 sniffImageMagic 的 webp 魔数一致）。
func isWebP(data []byte) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

// stripWebPMetadata 按 RIFF chunk 遍历删除 WebP 的元数据 chunk：
// "EXIF"（EXIF/GPS 载体）与 "XMP "（XMP 元数据，与 JPEG 删全部 APPn
// 的口径对齐）。VP8/VP8L/VP8X/ANMF/ALPH 等图像数据 chunk 原样保留，
// 不解码重编码（无质量损失、无新增依赖）。删除 chunk 后同步改写 RIFF
// 头的大小字段（文件总长-8）。chunk 边界按 RIFF 规范处理奇数字节的
// pad 字节（pad 不计入 chunk size，但属于容器布局，原样随邻块拷贝）。
func stripWebPMetadata(data []byte) ([]byte, bool, error) {
	var out bytes.Buffer
	out.Grow(len(data))
	out.Write(data[:12])
	pos := 12
	changed := false
	for pos+8 <= len(data) {
		chunkType := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		paddedEnd := pos + 8 + size
		if size%2 == 1 {
			paddedEnd++ // RIFF 奇数字节补 1 字节 pad
		}
		if paddedEnd > len(data) {
			return nil, false, errMalformedWebP
		}
		if chunkType == "EXIF" || chunkType == "XMP " {
			changed = true
		} else {
			out.Write(data[pos:paddedEnd])
		}
		pos = paddedEnd
	}
	if pos != len(data) {
		return nil, false, errMalformedWebP
	}
	result := out.Bytes()
	if changed {
		// RIFF 头 size 字段 = 文件总长 - 8（'RIFF' 四字节自身不计）；删除
		// chunk 后必须改写，否则容器声明长度与实际不符。未删除时保留原字节，
		// 维持「无修改即逐字节一致」的约定（畸形容器也不静默改写）。
		binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
	}
	return result, changed, nil
}
