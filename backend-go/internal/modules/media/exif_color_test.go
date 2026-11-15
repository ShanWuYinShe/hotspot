package media

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// jpegSegmentForTest 构造带长度字段的 JPEG 段（JPEG 段长含自身 2 字节）。
func jpegSegmentForTest(marker byte, payload []byte) []byte {
	out := []byte{0xFF, marker}
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(payload)+2))
	out = append(out, length...)
	return append(out, payload...)
}

// iccProfileSegment 构造 ICC_PROFILE APP2 段：ICC_PROFILE 头 + NUL 分隔符 +
// 段序号 + 段总数 + profile 片段（ICC 规范允许把大 profile 拆成多段）。
func iccProfileSegment(seq, total byte, profile string) []byte {
	payload := append([]byte("ICC_PROFILE"), 0)
	payload = append(payload, seq, total)
	return jpegSegmentForTest(0xE2, append(payload, []byte(profile)...))
}

// buildTestJPEGWithColorProfiles 构造含隐私元数据与色彩管理段的 JPEG：
// APP1（EXIF/GPS）、两段 APP2 ICC_PROFILE、一段非 ICC 的 APP2（FlashPix）、
// APP13（IPTC）、APP14（Adobe 色彩变换标记）。
func buildTestJPEGWithColorProfiles() []byte {
	out := []byte{0xFF, 0xD8}
	out = append(out, jpegSegmentForTest(0xE0, []byte("JFIF\x00\x01\x02\x00\x00\x01\x00\x01"))...)
	out = append(out, jpegSegmentForTest(0xE1, []byte("Exif GPS-GPS-GPS"))...)
	out = append(out, iccProfileSegment(1, 2, "profile-one")...)
	out = append(out, iccProfileSegment(2, 2, "profile-two")...)
	out = append(out, jpegSegmentForTest(0xE2, []byte("FPXR flashpix-meta"))...)
	out = append(out, jpegSegmentForTest(0xED, []byte("Photoshop 3.0 IPTC-CAPTION"))...)
	out = append(out, jpegSegmentForTest(0xEE, []byte("Adobe-COLOR"))...)
	out = append(out, jpegSegmentForTest(0xDB, []byte("\x00\x01\x02\x03"))...)
	out = append(out, jpegSegmentForTest(0xC0, []byte("\x08\x00\x01\x00\x01\x01"))...)
	out = append(out, jpegSegmentForTest(0xDA, []byte("\x01\x01\x00\x00\x3f\x00"))...)
	out = append(out, 0x12, 0x34, 0xFF, 0xD0, 0x56, 0x78, 0xFF, 0x00, 0x9A)
	out = append(out, 0xFF, 0xD9)
	return out
}

// jpegEntropyData 是 buildTestJPEG 系列使用的熵编码数据（含 RSTn 与 FF00 填充）。
var jpegEntropyData = []byte{0x12, 0x34, 0xFF, 0xD0, 0x56, 0x78, 0xFF, 0x00, 0x9A}

// 第 82 轮审计 P2-5：JPEG 清洗必须保留 ICC 色彩配置（APP2 ICC_PROFILE，可多段）
// 与 Adobe 标记（APP14），只删除 EXIF/XMP(APP1)、IPTC(APP13) 与其它元数据 APPn
// ——误删色彩管理会让广色域照片色偏、CMYK JPEG 反色；与 PNG 保留 iCCP 对齐。
func TestStripJPEGMetadataKeepsICCProfileAndAdobe(t *testing.T) {
	src := buildTestJPEGWithColorProfiles()
	cleaned, changed, err := stripImageMetadata(src)
	if err != nil {
		t.Fatalf("strip jpeg: %v", err)
	}
	if !changed {
		t.Fatal("expected EXIF/IPTC metadata segments to be removed")
	}
	// 隐私元数据必须消失（含非 ICC 的 APP2）。
	for _, payload := range []string{"GPS-GPS-GPS", "IPTC-CAPTION", "flashpix-meta"} {
		if bytes.Contains(cleaned, []byte(payload)) {
			t.Fatalf("metadata payload %q still present after scrub", payload)
		}
	}
	// 色彩管理数据必须保留：两段 ICC profile 与 Adobe 标记。
	for _, kept := range []string{"ICC_PROFILE", "profile-one", "profile-two", "Adobe-COLOR"} {
		if !bytes.Contains(cleaned, []byte(kept)) {
			t.Fatalf("color management payload %q must be preserved", kept)
		}
	}
	// ICC 多段按原顺序保留（段序号 1 在 2 之前）。
	first := bytes.Index(cleaned, []byte("profile-one"))
	second := bytes.Index(cleaned, []byte("profile-two"))
	if first < 0 || second < 0 || first > second {
		t.Fatalf("ICC segments must keep their order: index one=%d two=%d", first, second)
	}
	// JFIF 与熵编码数据原样保留，结构仍以 SOI 开头、EOI 结尾。
	if !bytes.Contains(cleaned, []byte("JFIF")) {
		t.Fatal("JFIF APP0 must be preserved")
	}
	if !bytes.Contains(cleaned, jpegEntropyData) {
		t.Fatal("entropy data must be preserved verbatim")
	}
	if cleaned[0] != 0xFF || cleaned[1] != 0xD8 || cleaned[len(cleaned)-2] != 0xFF || cleaned[len(cleaned)-1] != 0xD9 {
		t.Fatal("cleaned jpeg must start with SOI and end with EOI")
	}
	// 清洗幂等：再清洗一次无改动、逐字节一致。
	again, changedAgain, err := stripImageMetadata(cleaned)
	if err != nil || changedAgain || !bytes.Equal(again, cleaned) {
		t.Fatalf("second scrub must be noop: changed=%v err=%v", changedAgain, err)
	}
}

// 只有色彩管理段、没有隐私元数据时清洗必须逐字节不变（不能以「重排/重写」为名
// 改动字节，fuzz 不变量同样依赖该性质）。
func TestStripJPEGMetadataNoopWhenOnlyColorManagement(t *testing.T) {
	out := []byte{0xFF, 0xD8}
	out = append(out, jpegSegmentForTest(0xE0, []byte("JFIF\x00\x01\x02\x00\x00\x01\x00\x01"))...)
	out = append(out, iccProfileSegment(1, 1, "profile-only")...)
	out = append(out, jpegSegmentForTest(0xEE, []byte("Adobe-COLOR"))...)
	out = append(out, jpegSegmentForTest(0xDB, []byte("\x00\x01\x02\x03"))...)
	out = append(out, jpegSegmentForTest(0xC0, []byte("\x08\x00\x01\x00\x01\x01"))...)
	out = append(out, jpegSegmentForTest(0xDA, []byte("\x01\x01\x00\x00\x3f\x00"))...)
	out = append(out, jpegEntropyData...)
	out = append(out, 0xFF, 0xD9)

	cleaned, changed, err := stripImageMetadata(out)
	if err != nil || changed {
		t.Fatalf("jpeg with only color management must be noop: changed=%v err=%v", changed, err)
	}
	if !bytes.Equal(cleaned, out) {
		t.Fatal("clean jpeg must be byte-identical")
	}
}

// 非 ICC 的 APP2（FlashPix/MPF 等元数据）在没有其它元数据时也必须删除。
func TestStripJPEGMetadataRemovesNonICCAPP2(t *testing.T) {
	out := []byte{0xFF, 0xD8}
	out = append(out, jpegSegmentForTest(0xE0, []byte("JFIF\x00\x01\x02\x00\x00\x01\x00\x01"))...)
	out = append(out, jpegSegmentForTest(0xE2, []byte("FPXR flashpix-meta"))...)
	out = append(out, jpegSegmentForTest(0xDB, []byte("\x00\x01\x02\x03"))...)
	out = append(out, jpegSegmentForTest(0xC0, []byte("\x08\x00\x01\x00\x01\x01"))...)
	out = append(out, jpegSegmentForTest(0xDA, []byte("\x01\x01\x00\x00\x3f\x00"))...)
	out = append(out, jpegEntropyData...)
	out = append(out, 0xFF, 0xD9)

	cleaned, changed, err := stripImageMetadata(out)
	if err != nil || !changed {
		t.Fatalf("non-ICC APP2 must be removed: changed=%v err=%v", changed, err)
	}
	if bytes.Contains(cleaned, []byte("flashpix-meta")) {
		t.Fatal("FlashPix APP2 payload must be removed")
	}
	if !bytes.Contains(cleaned, []byte("JFIF")) {
		t.Fatal("JFIF APP0 must be preserved")
	}
}
