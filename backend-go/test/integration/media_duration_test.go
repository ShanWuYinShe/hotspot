package integration

import (
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
	"testing"
)

// 本文件构造「含 mvhd 的最小 mp4/m4a 字节」，用于验证 complete 阶段的服务端时长解析
// （internal/modules/media/duration_probe.go）：音频 moov 在头部、视频 moov 在尾部两种布局。

// ftypBox 生成最小 ftyp（isom）头。
func ftypBox() []byte {
	box := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	return append(box, make([]byte, 8)...)
}

// mvhdBox 生成 version 0 的 mvhd：type + version/flags + creation + modification + timescale + duration。
func mvhdBox(timescale, duration uint32) []byte {
	box := []byte{'m', 'v', 'h', 'd', 0x00, 0x00, 0x00, 0x00}
	box = append(box, make([]byte, 8)...) // creation + modification
	var word [4]byte
	binary.BigEndian.PutUint32(word[:], timescale)
	box = append(box, word[:]...)
	binary.BigEndian.PutUint32(word[:], duration)
	return append(box, word[:]...)
}

// moovBox 包一层 moov（内容仅 mvhd）。
func moovBox(mvhd []byte) []byte {
	payload := mvhd
	size := uint32(8 + len(payload))
	box := make([]byte, 8, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], size)
	copy(box[4:8], "moov")
	return append(box, payload...)
}

// uploadAndComplete 走完整链路（presign → PUT → complete）并返回 complete 的首个 upload 结果。
func uploadAndComplete(t *testing.T, fileName, mimeType, category string, payload []byte) map[string]any {
	t.Helper()
	var presignResp ApiResponse[map[string]any]
	apiRespOKWithHost(t, mediaPresignClientOrigin, "/api/v1/media/presign_upload", map[string]any{
		"files": []map[string]any{
			{"file_name": fileName, "file_size": len(payload), "mime_type": mimeType, "category": category},
		},
	}, &presignResp)
	entry := firstUploadEntry(t, presignResp.Data)
	uploadID, _ := entry["upload_id"].(string)
	uploadURL, _ := entry["upload_url"].(string)
	if uploadID == "" || uploadURL == "" {
		t.Fatalf("presign 缺少 upload_id/upload_url: %+v", presignResp)
	}

	putReq, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("创建 PUT 请求失败: %v", err)
	}
	putReq.Header.Set("Content-Type", mimeType)
	putResp, err := testClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT 对象失败: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("PUT 返回 %d: %s", putResp.StatusCode, body)
	}

	var completeResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/complete_upload", map[string]any{"upload_ids": []string{uploadID}}, &completeResp)
	if completeResp.Code != 0 || completeResp.Data == nil {
		t.Fatalf("complete_upload 失败: code=%d message=%s", completeResp.Code, completeResp.Message)
	}
	return firstUploadEntry(t, completeResp.Data)
}

// TestMedia_E2E_CompleteParsesAudioDuration：m4a（moov 在头部）complete 后返回 duration=2。
func TestMedia_E2E_CompleteParsesAudioDuration(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	payload := append(ftypBox(), moovBox(mvhdBox(1000, 2400))...)
	data := uploadAndComplete(t, "voice.m4a", "audio/aac", "chat_audio", payload)

	duration, ok := data["duration"].(float64)
	if !ok || int(duration) != 2 {
		t.Fatalf("期望 duration=2（2400/1000），实际 %v", data["duration"])
	}
}

// TestMedia_E2E_CompleteParsesVideoDurationFromTail：moov 位于文件尾（超过 64KB 前缀）时，
// complete 通过尾部 range 读取解析出 duration=13。
func TestMedia_E2E_CompleteParsesVideoDurationFromTail(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	payload := append(ftypBox(), bytes.Repeat([]byte{0x00}, 70*1024)...)
	payload = append(payload, moovBox(mvhdBox(1000, 13000))...)
	data := uploadAndComplete(t, "clip.mp4", "video/mp4", "chat_video", payload)

	duration, ok := data["duration"].(float64)
	if !ok || int(duration) != 13 {
		t.Fatalf("期望 duration=13（尾部 moov 13000/1000），实际 %v", data["duration"])
	}
}
