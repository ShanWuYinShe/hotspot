package media

import (
	"errors"
	"testing"
)

// audio/webm 是 Web 端 MediaRecorder 录制语音的默认容器（Opus）；契约分类表放开后
// 服务端必须接受，同时保持 wav 等白名单外格式拒绝。
func TestChatAudioWebmAllowedByContract(t *testing.T) {
	for _, mime := range []string{"audio/mpeg", "audio/aac", "audio/ogg", "audio/webm"} {
		if !isMimeAllowed(CategoryChatAudio, mime) {
			t.Fatalf("chat_audio 应接受 %q", mime)
		}
	}
	for _, mime := range []string{"audio/wav", "audio/mp4", "text/html"} {
		if isMimeAllowed(CategoryChatAudio, mime) {
			t.Fatalf("chat_audio 不应接受 %q", mime)
		}
	}
}

// 「扩展名 × MIME」双判定：.webm 已知且映射为 audio/webm，声明 video/webm 必须拒绝。
func TestValidateUploadFileContractWebmAudio(t *testing.T) {
	if err := validateUploadFileContract("voice.webm", "audio/webm"); err != nil {
		t.Fatalf("voice.webm + audio/webm 应通过，got %v", err)
	}
	if err := validateUploadFileContract("clip.webm", "video/webm"); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("clip.webm + video/webm 应拒绝，got %v", err)
	}
}

// 浏览器 MediaRecorder 上报的 mime 常带 codecs 参数，校验前须归一为 audio/webm。
func TestPresignChatAudioAcceptsCodecParameter(t *testing.T) {
	req := PresignFileInput{
		FileName: "voice.webm",
		FileSize: 100,
		MimeType: "audio/webm;codecs=opus",
		Category: CategoryChatAudio,
	}
	if err := callValidate(req); err != nil {
		t.Fatalf("audio/webm;codecs=opus 应通过归一后校验，got %v", err)
	}
}
