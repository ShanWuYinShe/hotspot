package media

import (
	"encoding/binary"
	"math"
)

// audioProbeHeadBytes：complete 阶段读取的对象前缀长度。WebM 的 Segment/Info
// （TimecodeScale + Duration）与 MP4 的 moov/mvhd 都位于文件前部，64KB 足够；
// 读不到就按「无时长」处理，不阻塞上传。
const audioProbeHeadBytes int64 = 64 * 1024

// audioProbeTailBytes：moov（MP4/MOV 时长索引）可能位于文件尾（非 faststart），
// 头部分解析失败时再读末尾这段重试；覆盖手机拍摄/剪辑导出的常见布局。
const audioProbeTailBytes int64 = 256 * 1024

// probeAudioDurationSeconds 从对象前缀解析媒体时长（秒，四舍五入）。
// 支持 WebM（MediaRecorder 录制语音）与 MP4/M4A（mvhd）；解析失败返回 ok=false，
// 由调用方保持 duration=null，客户端自行兜底（Web 端已有 seek 探测）。
func probeAudioDurationSeconds(mimeType string, head []byte) (int, bool) {
	switch normalizeMimeType(mimeType) {
	case "audio/webm", "video/webm":
		return probeWebMDuration(head)
	case "audio/mp4", "audio/aac", "video/mp4", "video/quicktime":
		return probeMP4Duration(head)
	default:
		return 0, false
	}
}

// probeDurationFromHeadAndTail 先按头部解析，失败再解析尾部（moov 在文件尾的 MP4）。
func probeDurationFromHeadAndTail(mimeType string, head, tail []byte) (int, bool) {
	if seconds, ok := probeAudioDurationSeconds(mimeType, head); ok {
		return seconds, true
	}
	if len(tail) == 0 {
		return 0, false
	}
	return probeAudioDurationSeconds(mimeType, tail)
}

// probeWebMDuration 解析 EBML：段信息里的 TimecodeScale（纳秒/单位，默认 1e6）
// 与 Duration（浮点，单位=TimecodeScale）。
func probeWebMDuration(head []byte) (int, bool) {
	scale := uint64(1_000_000)
	for i := 0; i+3 < len(head); i++ {
		// TimecodeScale：ID 0x2AD7B1，size 定长 vint，payload 为大端无符号整数
		if head[i] == 0x2A && head[i+1] == 0xD7 && head[i+2] == 0xB1 {
			if value, ok := readEBMLUint(head, i+3); ok {
				scale = value
			}
			continue
		}
		// Duration：ID 0x4489，payload 为 4 或 8 字节大端浮点
		if head[i] == 0x44 && head[i+1] == 0x89 {
			size, sizeLen, ok := readEBMLVint(head, i+2)
			if !ok || sizeLen <= 0 {
				continue
			}
			start := i + 2 + sizeLen
			end := start + int(size)
			if end > len(head) {
				continue
			}
			var units float64
			switch size {
			case 4:
				units = float64(math.Float32frombits(binary.BigEndian.Uint32(head[start:end])))
			case 8:
				units = math.Float64frombits(binary.BigEndian.Uint64(head[start:end]))
			default:
				continue
			}
			if units <= 0 || scale == 0 {
				return 0, false
			}
			seconds := int(math.Round(units * float64(scale) / 1e9))
			if seconds <= 0 {
				return 0, false
			}
			return seconds, true
		}
	}
	return 0, false
}

// probeMP4Duration 在文件前部查找 moov/mvhd，按 timescale/duration 计算秒数。
func probeMP4Duration(head []byte) (int, bool) {
	for i := 0; i+8 < len(head); i++ {
		if head[i] != 'm' || head[i+1] != 'v' || head[i+2] != 'h' || head[i+3] != 'd' {
			continue
		}
		payload := i + 4
		if payload+20 > len(head) {
			return 0, false
		}
		version := head[payload]
		var timescale, duration uint64
		if version == 1 {
			if payload+32 > len(head) {
				return 0, false
			}
			timescale = uint64(binary.BigEndian.Uint32(head[payload+20 : payload+24]))
			duration = binary.BigEndian.Uint64(head[payload+24 : payload+32])
		} else {
			timescale = uint64(binary.BigEndian.Uint32(head[payload+12 : payload+16]))
			duration = uint64(binary.BigEndian.Uint32(head[payload+16 : payload+20]))
		}
		if timescale == 0 || duration == 0 {
			return 0, false
		}
		seconds := int(math.Round(float64(duration) / float64(timescale)))
		if seconds <= 0 {
			return 0, false
		}
		return seconds, true
	}
	return 0, false
}

// readEBMLVint 读取 EBML 变长整数（保留 size 标记位：全 1 表示未知长度，调用方需跳过）。
func readEBMLVint(data []byte, offset int) (value uint64, length int, ok bool) {
	if offset >= len(data) {
		return 0, 0, false
	}
	first := data[offset]
	if first == 0 {
		return 0, 0, false
	}
	length = 1
	for mask := byte(0x80); mask > 0 && first&mask == 0; mask >>= 1 {
		length++
	}
	if length > 8 || offset+length > len(data) {
		return 0, 0, false
	}
	value = uint64(first) & (0xFF >> uint(length))
	for i := 1; i < length; i++ {
		value = value<<8 | uint64(data[offset+i])
	}
	return value, length, true
}

// readEBMLUint 读取「ID 之后的 size vint + 无符号整数 payload」组合。
func readEBMLUint(data []byte, offset int) (uint64, bool) {
	size, sizeLen, ok := readEBMLVint(data, offset)
	if !ok || sizeLen <= 0 || size == 0 || size > 8 {
		return 0, false
	}
	start := offset + sizeLen
	end := start + int(size)
	if end > len(data) {
		return 0, false
	}
	var value uint64
	for i := start; i < end; i++ {
		value = value<<8 | uint64(data[i])
	}
	return value, true
}
