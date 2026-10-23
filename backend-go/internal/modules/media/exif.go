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
// 保留 JFIF 所需的 APP0；SOS 之后是熵编码数据（内含 0xFFxx 填充与 RSTn
// 重启标记，不可按段解析），整段原样拷贝到 EOI。PNG 删除 eXIf 块。
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
func stripJPEGMetadata(data []byte) ([]byte, bool, error) {
	readSegmentLength := func(pos int) (int, error) {
		if pos+4 > len(data) {
			return 0, errMalformedJPEG
		}
		length := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if length < 2 || pos+2+length > len(data) {
			return 0, errMalformedJPEG
		}
		return length, nil
	}

	var out bytes.Buffer
	out.Grow(len(data))
	out.Write(data[:2]) // SOI
	pos := 2
	changed := false
	for pos < len(data) {
		if data[pos] != 0xFF || pos+1 >= len(data) {
			return nil, false, errMalformedJPEG
		}
		marker := data[pos+1]
		switch marker {
		case 0xD9: // EOI
			out.Write(data[pos:])
			return out.Bytes(), changed, nil
		case 0xDA: // SOS：其后为熵编码数据 + EOI，无更多可剥离段
			length, err := readSegmentLength(pos)
			if err != nil {
				return nil, false, err
			}
			out.Write(data[pos : pos+2+length])
			out.Write(data[pos+2+length:])
			return out.Bytes(), changed, nil
		}
		length, err := readSegmentLength(pos)
		if err != nil {
			return nil, false, err
		}
		if marker >= 0xE1 && marker <= 0xEF {
			changed = true // 丢弃 APP1~APP15（EXIF/XMP/IPTC/厂商注释）
		} else {
			out.Write(data[pos : pos+2+length])
		}
		pos += 2 + length
	}
	return out.Bytes(), changed, nil
}

// stripPNGMetadata 删除 PNG 的 eXIf 块（PNG 的 EXIF/GPS 载体），其余块
// 原样保留（长度/CRC 随块整体拷贝，不需重算）。
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
		if chunkType == "eXIf" {
			changed = true
		} else {
			out.Write(data[pos : pos+12+length])
		}
		pos = chunkEnd
	}
	if pos != len(data) {
		return nil, false, errMalformedPNG
	}
	return out.Bytes(), changed, nil
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
	// RIFF 头 size 字段 = 文件总长 - 8（'RIFF' 四字节自身不计）；删除
	// chunk 后必须改写，否则容器声明长度与实际不符。
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
	return result, changed, nil
}
