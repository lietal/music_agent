package tme

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/cipher"
	"crypto/des"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

var lyricsKey = []byte("!@#)(*$%123ZXC!@!@#)(NHL")

func (c *Client) GetLyrics(ctx context.Context, songID string) (*Lyrics, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallSong.PlayLyricInfo",
			Method: "GetPlayLyricInfo",
			Param: map[string]any{
				"songMid": mid,
				"crypt":   1,
				"trans":   1,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("lyrics failed: code=%d", sub.Code)
	}

	encryptedLyric := getString(sub.Data, "lyric")
	if encryptedLyric == "" {
		return &Lyrics{SongID: songID}, nil
	}

	decrypted, err := decryptLyrics(encryptedLyric)
	if err != nil {
		return &Lyrics{
			SongID:     songID,
			PlainText:  encryptedLyric,
			SyncedText: encryptedLyric,
		}, nil
	}

	transLyric := ""
	if transRaw := getString(sub.Data, "trans"); transRaw != "" {
		if dec, err := decryptLyrics(transRaw); err == nil {
			transLyric = dec
		}
	}

	fullLyric := decrypted
	if transLyric != "" {
		fullLyric += "\n\n" + transLyric
	}

	return &Lyrics{
		SongID:     songID,
		PlainText:  stripLRCTimestamps(fullLyric),
		SyncedText: fullLyric,
	}, nil
}

func decryptLyrics(encrypted string) (string, error) {
	data, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("hex decode: %w", err)
	}

	block, err := des.NewTripleDESCipher(lyricsKey)
	if err != nil {
		return "", fmt.Errorf("3des cipher: %w", err)
	}

	if len(data)%block.BlockSize() != 0 {
		return "", fmt.Errorf("invalid ciphertext length: %d", len(data))
	}

	mode := cipher.NewCBCDecrypter(block, make([]byte, block.BlockSize()))
	decrypted := make([]byte, len(data))
	mode.CryptBlocks(decrypted, data)

	decrypted = pkcs5Unpad(decrypted)

	reader, err := zlib.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		return "", fmt.Errorf("zlib reader: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return "", fmt.Errorf("zlib decompress: %w", err)
	}

	return buf.String(), nil
}

func pkcs5Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padLen := int(data[len(data)-1])
	if padLen > len(data) || padLen == 0 {
		return data
	}
	return data[:len(data)-padLen]
}

var lrcTimestampPattern = strings.NewReplacer

func stripLRCTimestamps(text string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		cleaned := trimLRC(trimmed)
		if cleaned != "" && !strings.HasPrefix(cleaned, "[") {
			result.WriteString(cleaned)
			result.WriteString("\n")
		}
	}
	return strings.TrimSpace(result.String())
}

func trimLRC(line string) string {
	for strings.HasPrefix(line, "[") {
		end := strings.Index(line, "]")
		if end == -1 {
			return line
		}
		line = line[end+1:]
	}
	return strings.TrimSpace(line)
}
