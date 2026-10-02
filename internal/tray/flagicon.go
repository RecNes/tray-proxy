package tray

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	flagCache   = map[string][]byte{}
	flagCacheMu sync.Mutex
	flagClient  = &http.Client{Timeout: 5 * time.Second}
)

// flagIconICO returns a small .ico for the ISO country code (cached).
// Windows tray menus cannot render emoji flags; they need bitmap icons.
func flagIconICO(code string) []byte {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) != 2 {
		return nil
	}
	flagCacheMu.Lock()
	if b, ok := flagCache[code]; ok {
		flagCacheMu.Unlock()
		return b
	}
	flagCacheMu.Unlock()

	ico := fetchFlagICO(code)
	if ico == nil {
		ico = placeholderFlagICO(code)
	}
	flagCacheMu.Lock()
	flagCache[code] = ico
	flagCacheMu.Unlock()
	return ico
}

func fetchFlagICO(code string) []byte {
	url := fmt.Sprintf("https://flagcdn.com/16x12/%s.png", code)
	resp, err := flagClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil
	}
	ico, err := pngBytesToICO(buf.Bytes())
	if err != nil {
		return nil
	}
	return ico
}

func placeholderFlagICO(code string) []byte {
	const w, h = 16, 12
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if y < h/2 {
				img.Set(x, y, image.White)
			} else {
				img.Set(x, y, image.Black)
			}
		}
	}
	_ = code
	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, img)
	ico, err := pngBytesToICO(pngBuf.Bytes())
	if err != nil {
		return nil
	}
	return ico
}

func pngBytesToICO(pngBytes []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(b)
		draw.Draw(rgba, b, img, b.Min, draw.Src)
	}

	// XOR bitmap: bottom-up BGRA
	xor := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		for x := 0; x < w; x++ {
			i := (srcY*w + x) * 4
			o := (y*w + x) * 4
			xor[o+0] = rgba.Pix[i+2] // B
			xor[o+1] = rgba.Pix[i+1] // G
			xor[o+2] = rgba.Pix[i+0] // R
			xor[o+3] = rgba.Pix[i+3] // A
		}
	}
	// AND mask: 1 bit/pixel, rows padded to 32 bits
	rowBytes := ((w + 31) / 32) * 4
	andMask := make([]byte, rowBytes*h)

	dibSize := 40 + len(xor) + len(andMask)
	var dib bytes.Buffer
	_ = binary.Write(&dib, binary.LittleEndian, uint32(40))
	_ = binary.Write(&dib, binary.LittleEndian, int32(w))
	_ = binary.Write(&dib, binary.LittleEndian, int32(h*2))
	_ = binary.Write(&dib, binary.LittleEndian, uint16(1))
	_ = binary.Write(&dib, binary.LittleEndian, uint16(32))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(0))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(len(xor)+len(andMask)))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(0))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(0))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(0))
	_ = binary.Write(&dib, binary.LittleEndian, uint32(0))
	dib.Write(xor)
	dib.Write(andMask)

	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // count
	out.WriteByte(byte(w % 256))
	out.WriteByte(byte(h % 256))
	out.WriteByte(0) // colors
	out.WriteByte(0) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))  // planes
	_ = binary.Write(&out, binary.LittleEndian, uint16(32)) // bitcount
	_ = binary.Write(&out, binary.LittleEndian, uint32(dibSize))
	_ = binary.Write(&out, binary.LittleEndian, uint32(6+16)) // offset
	out.Write(dib.Bytes())
	return out.Bytes(), nil
}
