package handlers

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"devtools/config"
	"devtools/models"
)

// 本文件覆盖 Stage 3 的「上游直链泛型 MIME 归一化」实现（backend/handlers/minimax_result_share.go）。
//
// 断言只使用「改动前后都存在」的符号（readMiniMaxShareAssetContent / persistMiniMaxResultShareAssets /
// miniMaxShareAssetKind 等），因此同一份测试可以直接对着改动前的 main 跑一次做反事实自检：
// 基线上「泛型 MIME → 真实媒体类型」的用例会失败，被 FAIL 版能命中失败断言，修复分支全绿。
//
// 字节夹具说明（依据 Go 标准库 net/http/sniff.go 的实测签名表）：
//   - 带 ID3 头的字节 → DetectContentType 返回 audio/mpeg（maskedSig "ID3"）。
//   - PNG 头 \x89PNG\r\n\x1a\n → DetectContentType 返回 image/png。
//   - 既非文本、又不命中任何嗅探签名的二进制 → 嗅探回落为 application/octet-stream（泛型），
//     只有这种字节才会走到「按 URL 后缀推断」的分支。

// miniMaxShareMIMEHandler 构造带 :memory: DB 的 AIGatewayHandler。
// R5：:memory: DB 必须 SetMaxOpenConns(1)，否则后台 goroutine 可能落到不同连接上。
func miniMaxShareMIMEHandler(t *testing.T) (*AIGatewayHandler, *models.DB) {
	t.Helper()
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitMiniMaxResultShares(); err != nil {
		t.Fatalf("init result shares: %v", err)
	}
	return NewAIGatewayHandler(db, config.DefaultConfig(), nil, testEncEncryptionService), db
}

// miniMaxShareMP3Bytes 返回带 ID3 头的真实 MP3 字节（嗅探结果 audio/mpeg）。
func miniMaxShareMP3Bytes() []byte {
	return []byte{
		'I', 'D', '3', 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0xFB, 0x90, 0x00, 0x00,
		0x21, 0x00, 0x00, 0x00, 0x00,
	}
}

// miniMaxSharePNGBytes 返回真实 PNG 头（嗅探结果 image/png）。
func miniMaxSharePNGBytes() []byte {
	return []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
}

// miniMaxShareOpaqueBytes 返回既不可打印文本、也不命中任何嗅探签名的二进制，
// DetectContentType 对它回落为 application/octet-stream（泛型 MIME）。
func miniMaxShareOpaqueBytes() []byte {
	return []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xF1}
}

// miniMaxShareWAVBytes 返回 RIFF/WAVE 头，http.DetectContentType 对它返回 audio/wave
// （注意：与内置后缀表里的 audio/wav 不同名，sniffed 值先于后缀表返回）。
func miniMaxShareWAVBytes() []byte {
	return []byte{
		'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00,
		'W', 'A', 'V', 'E', 'f', 'm', 't', ' ',
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		'd', 'a', 't', 'a', 0x00, 0x00, 0x00, 0x00,
	}
}

// TestMiniMaxShareAssetGenericMINEIsNormalized 覆盖泛型 MIME 的三条归一化路径
// （嗅探 → URL 后缀 → 保留原响应头）以及既有 MIME 的回归。
func TestMiniMaxShareAssetGenericMIMEIsNormalized(t *testing.T) {
	cases := []struct {
		name            string
		headerType      string
		suppressType    bool
		body            []byte
		path            string
		preferredName   string
		wantContentType string
		wantKind        string
		wantFilename    string
	}{
		{
			name:            "泛型 MIME + MP3 字节 + .mp3 后缀 → 归一化为 audio/mpeg",
			headerType:      "application/octet-stream",
			body:            miniMaxShareMP3Bytes(),
			path:            "/voice.mp3",
			wantContentType: "audio/mpeg",
			wantKind:        "audio",
			wantFilename:    "asset_01.mp3",
		},
		{
			name:            "泛型 MIME + MP3 字节 + 无后缀 URL → 靠字节嗅探得到 audio/mpeg",
			headerType:      "application/octet-stream",
			body:            miniMaxShareMP3Bytes(),
			path:            "/blob",
			wantContentType: "audio/mpeg",
			wantKind:        "audio",
			wantFilename:    "asset_01.mp3",
		},
		{
			name:            "泛型 MIME + PNG 字节 + 无后缀 URL → 靠字节嗅探得到 image/png",
			headerType:      "application/octet-stream",
			body:            miniMaxSharePNGBytes(),
			path:            "/blob",
			wantContentType: "image/png",
			wantKind:        "image",
			wantFilename:    "asset_01.png",
		},
		{
			name:            "泛型 MIME + 嗅探不出的二进制 + .aac 后缀 → 靠后缀得到 audio/aac",
			headerType:      "application/octet-stream",
			body:            miniMaxShareOpaqueBytes(),
			path:            "/voice.aac",
			wantContentType: "audio/aac",
			wantKind:        "audio",
			wantFilename:    "asset_01.aac",
		},
		{
			name:            "泛型 MIME + 嗅探不出的二进制 + .flac 后缀 → 靠后缀得到 audio/flac",
			headerType:      "application/octet-stream",
			body:            miniMaxShareOpaqueBytes(),
			path:            "/voice.flac",
			wantContentType: "audio/flac",
			wantKind:        "audio",
			wantFilename:    "asset_01.flac",
		},
		{
			name:            "binary/octet-stream 同样按泛型处理 + .m4a 后缀 → audio/mp4",
			headerType:      "binary/octet-stream",
			body:            miniMaxShareOpaqueBytes(),
			path:            "/voice.m4a",
			wantContentType: "audio/mp4",
			wantKind:        "audio",
			wantFilename:    "asset_01.m4a",
		},
		{
			name:            "泛型 MIME + 无后缀 URL + 嗅探不出的二进制 → 保留原响应头",
			headerType:      "application/octet-stream",
			body:            miniMaxShareOpaqueBytes(),
			path:            "/blob",
			preferredName:   "blob.bin",
			wantContentType: "application/octet-stream",
			wantKind:        "file",
			wantFilename:    "asset_01.bin",
		},
		{
			name:            "回归：audio/mpeg 响应头保持不变",
			headerType:      "audio/mpeg",
			body:            miniMaxShareMP3Bytes(),
			path:            "/voice.mp3",
			wantContentType: "audio/mpeg",
			wantKind:        "audio",
			wantFilename:    "asset_01.mp3",
		},
		{
			name:            "回归：带 charset 参数的 audio/mpeg 仍归一为 audio/mpeg",
			headerType:      "audio/mpeg; charset=binary",
			body:            miniMaxShareMP3Bytes(),
			path:            "/voice.mp3",
			wantContentType: "audio/mpeg",
			wantKind:        "audio",
			wantFilename:    "asset_01.mp3",
		},
		{
			name:            "回归：上游完全没有 Content-Type → 靠字节嗅探兜底",
			headerType:      "",
			suppressType:    true,
			body:            miniMaxShareMP3Bytes(),
			path:            "/voice.mp3",
			wantContentType: "audio/mpeg",
			wantKind:        "audio",
			wantFilename:    "asset_01.mp3",
		},
		{
			// Go 嗅探器对 RIFF....WAVE 返回 audio/wave，与内置后缀表里的 audio/wav 不同名
			// （sniffed 值先于后缀表返回）。这里只断言 kind 与扩展名，不断言精确 content_type。
			name:            "RIFF/WAVE 字节 → 嗅探出 audio 系，扩展名仍是 .wav",
			headerType:      "application/octet-stream",
			body:            miniMaxShareWAVBytes(),
			path:            "/voice.wav",
			wantContentType: "audio/wave",
			wantKind:        "audio",
			wantFilename:    "asset_01.wav",
		},
		{
			name:            "URL 带 query 参数时后缀推断仍只看路径",
			headerType:      "application/octet-stream",
			body:            miniMaxShareOpaqueBytes(),
			path:            "/voice.aac?token=abc123&t=1",
			wantContentType: "audio/aac",
			wantKind:        "audio",
			wantFilename:    "asset_01.aac",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 资产落盘目录是相对路径（data/minimax_result_shares/...），
			// 切到临时目录避免在仓库盘上产生文件。
			t.Chdir(t.TempDir())

			h, db := miniMaxShareMIMEHandler(t)
			defer db.Close()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.suppressType {
					// 显式置 nil 让 net/http 不再自动嗅探 Content-Type。
					w.Header()["Content-Type"] = nil
				} else {
					w.Header().Set("Content-Type", tc.headerType)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tc.body)
			}))
			defer srv.Close()

			input := MiniMaxResultShareAssetInput{
				URL:      srv.URL + tc.path,
				Filename: tc.preferredName,
			}
			shareID := "mrs_mime_normalize"

			// 1) 下载 + 归一化阶段
			data, contentType, err := h.readMiniMaxShareAssetContent(input)
			if err != nil {
				t.Fatalf("read asset content: %v", err)
			}
			if contentType != tc.wantContentType {
				t.Fatalf("content_type = %q, want %q", contentType, tc.wantContentType)
			}
			if len(data) != len(tc.body) {
				t.Fatalf("downloaded %d bytes, want %d", len(data), len(tc.body))
			}

			// 2) 落盘 + 生成资产元信息
			assets, err := h.persistMiniMaxResultShareAssets(shareID, []MiniMaxResultShareAssetInput{input})
			if err != nil {
				t.Fatalf("persist assets: %v", err)
			}
			if len(assets) != 1 {
				t.Fatalf("assets count = %d, want 1", len(assets))
			}
			asset := assets[0]
			if asset.ContentType != tc.wantContentType {
				t.Fatalf("asset.ContentType = %q, want %q", asset.ContentType, tc.wantContentType)
			}
			if asset.Kind != tc.wantKind {
				t.Fatalf("asset.Kind = %q, want %q", asset.Kind, tc.wantKind)
			}
			if asset.Filename != tc.wantFilename {
				t.Fatalf("asset.Filename = %q, want %q", asset.Filename, tc.wantFilename)
			}
			if asset.SizeBytes != int64(len(tc.body)) {
				t.Fatalf("asset.SizeBytes = %d, want %d", asset.SizeBytes, len(tc.body))
			}

			// 3) 落库并回读，验证 content_type / kind 真的写进了 SQLite
			share := &models.MiniMaxResultShare{
				ID:         shareID,
				ResultType: "media",
				Payload:    "{}",
				AssetsJSON: models.MustMarshalMiniMaxResultShareAssets(assets),
			}
			if err := db.CreateMiniMaxResultShare(share); err != nil {
				t.Fatalf("create share: %v", err)
			}
			stored, err := db.GetMiniMaxResultShare(shareID)
			if err != nil {
				t.Fatalf("get share: %v", err)
			}
			storedAssets, err := models.ParseMiniMaxResultShareAssets(stored.AssetsJSON)
			if err != nil {
				t.Fatalf("parse stored assets: %v", err)
			}
			if len(storedAssets) != 1 {
				t.Fatalf("stored assets count = %d, want 1", len(storedAssets))
			}
			if storedAssets[0].ContentType != tc.wantContentType {
				t.Fatalf("stored content_type = %q, want %q", storedAssets[0].ContentType, tc.wantContentType)
			}
			if storedAssets[0].Kind != tc.wantKind {
				t.Fatalf("stored kind = %q, want %q", storedAssets[0].Kind, tc.wantKind)
			}
			if storedAssets[0].Filename != tc.wantFilename {
				t.Fatalf("stored filename = %q, want %q", storedAssets[0].Filename, tc.wantFilename)
			}

			// 4) 资产文件确实落盘
			assetPath := filepath.Join(h.minimaxResultShareDir(shareID), asset.Filename)
			info, err := os.Stat(assetPath)
			if err != nil {
				t.Fatalf("asset file missing: %v", err)
			}
			if info.Size() != int64(len(tc.body)) {
				t.Fatalf("asset file size = %d, want %d", info.Size(), len(tc.body))
			}
		})
	}
}

// TestMiniMaxShareAssetDataURLBranchUnaffected 验证 data_url 分支不被泛型 MIME 归一化影响：
// data URL 头部的 MIME 仍然说了算。
func TestMiniMaxShareAssetDataURLBranchUnaffected(t *testing.T) {
	cases := []struct {
		name            string
		dataURL         string
		preferredName   string
		wantContentType string
		wantKind        string
		wantFilename    string
	}{
		{
			name:            "data URL 头部的 image/png 仍然生效",
			dataURL:         "data:image/png;base64," + base64.StdEncoding.EncodeToString(miniMaxSharePNGBytes()),
			preferredName:   "shot.png",
			wantContentType: "image/png",
			wantKind:        "image",
			wantFilename:    "asset_01.png",
		},
		{
			name:            "data URL 头部是泛型 MIME 时不做归一化（保持原值）",
			dataURL:         "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(miniMaxShareOpaqueBytes()),
			preferredName:   "blob.bin",
			wantContentType: "application/octet-stream",
			wantKind:        "file",
			wantFilename:    "asset_01.bin",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			h, db := miniMaxShareMIMEHandler(t)
			defer db.Close()

			input := MiniMaxResultShareAssetInput{
				DataURL:  tc.dataURL,
				Filename: tc.preferredName,
			}
			shareID := "mrs_mime_data_url"

			data, contentType, err := h.readMiniMaxShareAssetContent(input)
			if err != nil {
				t.Fatalf("read asset content: %v", err)
			}
			if contentType != tc.wantContentType {
				t.Fatalf("content_type = %q, want %q", contentType, tc.wantContentType)
			}
			if len(data) == 0 {
				t.Fatalf("decoded data is empty")
			}

			assets, err := h.persistMiniMaxResultShareAssets(shareID, []MiniMaxResultShareAssetInput{input})
			if err != nil {
				t.Fatalf("persist assets: %v", err)
			}
			if len(assets) != 1 {
				t.Fatalf("assets count = %d, want 1", len(assets))
			}
			if assets[0].ContentType != tc.wantContentType {
				t.Fatalf("asset.ContentType = %q, want %q", assets[0].ContentType, tc.wantContentType)
			}
			if assets[0].Kind != tc.wantKind {
				t.Fatalf("asset.Kind = %q, want %q", assets[0].Kind, tc.wantKind)
			}
			if assets[0].Filename != tc.wantFilename {
				t.Fatalf("asset.Filename = %q, want %q", assets[0].Filename, tc.wantFilename)
			}
		})
	}
}
