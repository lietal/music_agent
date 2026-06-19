package tme

import (
	"bytes"
	"compress/zlib"
	"crypto/cipher"
	"crypto/des"
	"encoding/hex"
	"testing"
)

func TestDecryptLyrics_HappyPath(t *testing.T) {
	plain := "hello world 周杰伦"
	encrypted := encryptForTest(t, plain)
	dec, err := decryptLyrics(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if dec != plain {
		t.Errorf("got %q", dec)
	}
}

func encryptForTest(t *testing.T, plain string) string {
	t.Helper()
	var zbuf bytes.Buffer
	w := zlib.NewWriter(&zbuf)
	w.Write([]byte(plain))
	w.Close()
	zdata := zbuf.Bytes()

	block, err := des.NewTripleDESCipher(lyricsKey)
	if err != nil {
		t.Fatal(err)
	}
	bs := block.BlockSize()
	padLen := bs - (len(zdata) % bs)
	padded := make([]byte, len(zdata)+padLen)
	copy(padded, zdata)
	for i := len(zdata); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	mode := cipher.NewCBCEncrypter(block, make([]byte, bs))
	enc := make([]byte, len(padded))
	mode.CryptBlocks(enc, padded)
	return hex.EncodeToString(enc)
}
