package media

import (
	"encoding/binary"
	"math"
	"testing"
)

func ebmlWebMFixture(scale uint64, durationunits float64) []byte {
	buffer := []byte{0x1A, 0x45, 0xDF, 0xA3} // 段头前缀，解析器应能跳过
	// TimecodeScale：ID + 3 字节 size + 3 字节无符号数
	buffer = append(buffer, 0x2A, 0xD7, 0xB1, 0x83,
		byte(scale>>16), byte(scale>>8), byte(scale))
	// Duration：ID + 8 字节 size + float64
	buffer = append(buffer, 0x44, 0x89, 0x88)
	bits := math.Float64bits(durationunits)
	for shift := 56; shift >= 0; shift -= 8 {
		buffer = append(buffer, byte(bits>>uint(shift)))
	}
	return buffer
}

func mp4Fixture(timescale, duration uint32) []byte {
	buffer := []byte{0x00, 0x00, 0x00, 0x6D, 'm', 'v', 'h', 'd', 0x00, 0x00, 0x00, 0x00}
	for i := 0; i < 8; i++ { // creation + modification
		buffer = append(buffer, 0)
	}
	var word [4]byte
	binary.BigEndian.PutUint32(word[:], timescale)
	buffer = append(buffer, word[:]...)
	binary.BigEndian.PutUint32(word[:], duration)
	buffer = append(buffer, word[:]...)
	return buffer
}

func TestProbeAudioDurationWebM(t *testing.T) {
	seconds, ok := probeAudioDurationSeconds("audio/webm", ebmlWebMFixture(1_000_000, 7400))
	if !ok || seconds != 7 {
		t.Fatalf("webm 7.4s 应解析为 7s，got %d ok=%v", seconds, ok)
	}
	if _, ok := probeAudioDurationSeconds("audio/webm", []byte{0x44, 0x89}); ok {
		t.Fatal("截断的 webm 不应返回时长")
	}
}

func TestProbeAudioDurationMP4(t *testing.T) {
	seconds, ok := probeAudioDurationSeconds("audio/aac", mp4Fixture(1000, 2400))
	if !ok || seconds != 2 {
		t.Fatalf("mvhd 2400/1000 应解析为 2s，got %d ok=%v", seconds, ok)
	}
	if _, ok := probeAudioDurationSeconds("audio/aac", mp4Fixture(0, 2400)); ok {
		t.Fatal("timescale=0 应拒绝")
	}
}

func TestProbeAudioDurationUnsupported(t *testing.T) {
	for _, mime := range []string{"audio/mpeg", "image/png", "text/plain"} {
		if _, ok := probeAudioDurationSeconds(mime, ebmlWebMFixture(1_000_000, 1000)); ok {
			t.Fatalf("%s 不应走音频时长解析", mime)
		}
	}
	if seconds, ok := probeAudioDurationSeconds("audio/webm", nil); ok || seconds != 0 {
		t.Fatal("空数据应返回 false")
	}
}

func TestProbeDurationFromHeadAndTail(t *testing.T) {
	headWithoutMoov := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	tailWithMoov := mp4Fixture(1000, 13000)
	seconds, ok := probeDurationFromHeadAndTail("video/mp4", headWithoutMoov, tailWithMoov)
	if !ok || seconds != 13 {
		t.Fatalf("moov 在尾部时应解析出 13s，got %d ok=%v", seconds, ok)
	}
	// 头部已有 moov（faststart）时直接命中，无需尾部
	seconds, ok = probeDurationFromHeadAndTail("video/mp4", mp4Fixture(1000, 5000), nil)
	if !ok || seconds != 5 {
		t.Fatalf("faststart 应解析出 5s，got %d ok=%v", seconds, ok)
	}
}
